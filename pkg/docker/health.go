package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/sameerchandra/piman/pkg/config"
	"gopkg.in/yaml.v3"
)

// Deployment represents an app mapping to a node from the deployments file
type Deployment struct {
	Node string `yaml:"node"`
	App  string `yaml:"app"`
}

// DeploymentsConfig represents the schema of deployments.yaml
type DeploymentsConfig struct {
	Deployments []Deployment `yaml:"deployments"`
}

// ServiceStatus represents the health of a single compose service
type ServiceStatus struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Status  string `json:"status"` // e.g. "running", "exited", "missing"
}

// AppStatus represents the health of an entire application
type AppStatus struct {
	AppName  string          `json:"app_name"`
	Health   string          `json:"health"` // "Healthy", "Degraded", "Stopped"
	Services []ServiceStatus `json:"services"`
}

// GetDeployments loads desired deployments from catalog cache state files
func GetDeployments() ([]Deployment, error) {
	pistoreDir, err := config.GetPistoreDir()
	if err != nil {
		return nil, err
	}

	// We support either deployments.yaml or installed.yaml inside the state folder
	paths := []string{
		filepath.Join(pistoreDir, "catalog_cache", "state", "deployments.yaml"),
		filepath.Join(pistoreDir, "catalog_cache", "state", "installed.yaml"),
	}

	var data []byte
	var pathFound string
	for _, p := range paths {
		if b, err := os.ReadFile(p); err == nil {
			data = b
			pathFound = p
			break
		}
	}

	if data == nil {
		// If deployments/installed state file is not found, degrade gracefully by returning an empty list
		return nil, nil
	}

	var cfg DeploymentsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filepath.Base(pathFound), err)
	}

	return cfg.Deployments, nil
}

// GetNodeStatus connects to the remote node, fetches actual container states,
// and cross-references them against deployments and compose files to compute app health.
func GetNodeStatus(nodeName string) ([]AppStatus, error) {
	// 1. Load desired deployments
	deployments, err := GetDeployments()
	if err != nil {
		return nil, err
	}

	// Filter deployments for target node
	var nodeApps []string
	for _, d := range deployments {
		if d.Node == nodeName {
			nodeApps = append(nodeApps, d.App)
		}
	}

	// Proceed to query remote node status regardless of whether there are configured catalog apps.
	// If nodeApps is empty, it will simply skip the catalog cross-reference and list standalone containers.

	// 2. Instantiate remote Docker client over SSH
	cli, sshClient, err := NewRemoteDockerClient(nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote Docker: %w", err)
	}
	defer sshClient.Close()
	defer cli.Close()

	// 3. List actual containers (including stopped ones)
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list remote containers: %w", err)
	}

	pistoreDir, err := config.GetPistoreDir()
	if err != nil {
		return nil, err
	}

	var results []AppStatus
	matchedIDs := make(map[string]bool)

	// 4. Evaluate each application's health
	for _, appName := range nodeApps {
		composePath := filepath.Join(pistoreDir, "catalog_cache", "catalog", appName, "docker-compose.yml")

		// Read compose file to find defined services
		data, err := os.ReadFile(composePath)
		if err != nil {
			results = append(results, AppStatus{
				AppName: appName,
				Health:  "Stopped (Compose missing)",
			})
			continue
		}

		var cmp struct {
			Services map[string]interface{} `yaml:"services"`
		}
		if err := yaml.Unmarshal(data, &cmp); err != nil {
			results = append(results, AppStatus{
				AppName: appName,
				Health:  "Error parsing compose",
			})
			continue
		}

		var services []ServiceStatus
		runningCount := 0

		for svcName := range cmp.Services {
			svcStatus := ServiceStatus{
				Name:    svcName,
				Running: false,
				Status:  "missing",
			}

			// Match containers by compose project and service labels
			for _, container := range containers {
				projLabel := container.Labels["com.docker.compose.project"]
				svcLabel := container.Labels["com.docker.compose.service"]

				if projLabel == appName && svcLabel == svcName {
					svcStatus.Status = container.State
					if container.State == "running" {
						svcStatus.Running = true
						runningCount++
					}
					matchedIDs[container.ID] = true
					break
				}
			}

			services = append(services, svcStatus)
		}

		// Calculate overall health
		health := "Stopped"
		if len(services) > 0 {
			if runningCount == len(services) {
				health = "Healthy"
			} else if runningCount > 0 {
				health = "Degraded"
			}
		}

		results = append(results, AppStatus{
			AppName:  appName,
			Health:   health,
			Services: services,
		})
	}

	// 5. Gather unmanaged/standalone containers as separate apps
	for _, container := range containers {
		if !matchedIDs[container.ID] {
			name := "unknown"
			if len(container.Names) > 0 {
				name = strings.TrimPrefix(container.Names[0], "/")
			}
			running := container.State == "running"
			health := "Stopped"
			if running {
				health = "Healthy"
			}

			results = append(results, AppStatus{
				AppName: name,
				Health:  health,
				Services: []ServiceStatus{
					{
						Name:    name,
						Running: running,
						Status:  container.State,
					},
				},
			})
		}
	}

	return results, nil
}
