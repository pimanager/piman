package router

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/sameerchandra/piman/pkg/config"
	"github.com/sameerchandra/piman/pkg/docker"
	"gopkg.in/yaml.v3"
)

// Route represents a mapping from domain to a node & port in the cluster
type Route struct {
	Domain         string `yaml:"domain" json:"domain"`
	Node           string `yaml:"node" json:"node"`
	Port           int    `yaml:"port" json:"port"`
	Service        string `yaml:"service,omitempty" json:"service,omitempty"`
	AutoDiscovered bool   `yaml:"-" json:"auto_discovered"`
}

// RoutesConfig represents the structure of ~/.pistore/routes.yaml
type RoutesConfig struct {
	Routes []Route `yaml:"routes"`
}

// GetRoutes returns the combined list of custom and auto-discovered routes
func GetRoutes() ([]Route, error) {
	customRoutes, err := LoadCustomRoutes()
	if err != nil {
		customRoutes = []Route{}
	}

	discoveredRoutes, err := DiscoverRoutes()
	if err != nil {
		discoveredRoutes = []Route{}
	}

	// Combine them, letting custom routes override auto-discovered ones for the same domain
	routesMap := make(map[string]Route)
	for _, r := range discoveredRoutes {
		routesMap[r.Domain] = r
	}
	for _, r := range customRoutes {
		// Custom route takes precedence
		routesMap[r.Domain] = r
	}

	var combined []Route
	for _, r := range routesMap {
		combined = append(combined, r)
	}

	return combined, nil
}

// LoadCustomRoutes loads custom routes from ~/.pistore/routes.yaml
func LoadCustomRoutes() ([]Route, error) {
	pDir, err := config.GetPistoreDir()
	if err != nil {
		return nil, err
	}

	routesPath := filepath.Join(pDir, "routes.yaml")
	data, err := os.ReadFile(routesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Route{}, nil
		}
		return nil, err
	}

	var cfg RoutesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse routes.yaml: %w", err)
	}

	return cfg.Routes, nil
}

// SaveCustomRoutes saves custom routes list to ~/.pistore/routes.yaml
func SaveCustomRoutes(routes []Route) error {
	pDir, err := config.GetPistoreDir()
	if err != nil {
		return err
	}

	// Filter out auto-discovered routes from saving
	var custom []Route
	for _, r := range routes {
		if !r.AutoDiscovered {
			custom = append(custom, r)
		}
	}

	cfg := RoutesConfig{Routes: custom}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	routesPath := filepath.Join(pDir, "routes.yaml")
	return os.WriteFile(routesPath, data, 0644)
}

// DiscoverRoutes scans all nodes to auto-discover running services and their port mappings
func DiscoverRoutes() ([]Route, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	var routes []Route

	for _, node := range cfg.Nodes {
		// 1. Establish connection to node's docker daemon
		cli, sshClient, err := docker.NewRemoteDockerClient(node.Name)
		if err != nil {
			// Skip node if unreachable
			continue
		}

		// 2. List containers (running only)
		containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: false})
		if err != nil {
			sshClient.Close()
			cli.Close()
			continue
		}

		for _, c := range containers {
			for _, port := range c.Ports {
				// We map if there is a public port exposed
				if port.PublicPort != 0 {
					appName := c.Labels["com.docker.compose.project"]
					svcName := c.Labels["com.docker.compose.service"]

					if appName == "" {
						// Standalone container
						if len(c.Names) > 0 {
							appName = strings.TrimPrefix(c.Names[0], "/")
						} else {
							appName = c.ID[:12]
						}
						svcName = appName
					}

					// Domain: <app>.<node>.piman.local or just <app>.local
					domain := fmt.Sprintf("%s.local", appName)
					if svcName != appName {
						domain = fmt.Sprintf("%s-%s.local", appName, svcName)
					}

					routes = append(routes, Route{
						Domain:         domain,
						Node:           node.Name,
						Port:           int(port.PublicPort),
						Service:        svcName,
						AutoDiscovered: true,
					})
				}
			}
		}

		sshClient.Close()
		cli.Close()
	}

	return routes, nil
}

// EnsureConfigFilesExist creates the base ~/.pistore/router/nginx.conf if missing
func EnsureConfigFilesExist() error {
	pDir, err := config.GetPistoreDir()
	if err != nil {
		return err
	}

	routerDir := filepath.Join(pDir, "router")
	if err := os.MkdirAll(filepath.Join(routerDir, "conf.d"), 0755); err != nil {
		return err
	}

	nginxConfPath := filepath.Join(routerDir, "nginx.conf")
	if _, err := os.Stat(nginxConfPath); os.IsNotExist(err) {
		nginxConfContent := `user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;

    sendfile on;
    keepalive_timeout 65;

    include /etc/nginx/conf.d/*.conf;
}
`
		if err := os.WriteFile(nginxConfPath, []byte(nginxConfContent), 0644); err != nil {
			return err
		}
	}

	// Create mime.types locally if it does not exist (in case host fallback needs it)
	mimePath := filepath.Join(routerDir, "mime.types")
	if _, err := os.Stat(mimePath); os.IsNotExist(err) {
		mimeContent := `types {
    text/html                             html htm shtml;
    text/css                              css;
    text/xml                              xml;
    image/gif                             gif;
    image/jpeg                            jpeg jpg;
    application/javascript                js;
    application/json                      json;
    image/png                             png;
    image/x-icon                          ico;
}
`
		_ = os.WriteFile(mimePath, []byte(mimeContent), 0644)
	}

	return nil
}

// RegenerateNginxConfig re-writes conf.d configurations based on combined routes list
func RegenerateNginxConfig() error {
	if err := EnsureConfigFilesExist(); err != nil {
		return err
	}

	routes, err := GetRoutes()
	if err != nil {
		return fmt.Errorf("failed to gather routes: %w", err)
	}

	pDir, err := config.GetPistoreDir()
	if err != nil {
		return err
	}
	confDDir := filepath.Join(pDir, "router", "conf.d")

	// Clean out existing files
	_ = os.RemoveAll(confDDir)
	if err := os.MkdirAll(confDDir, 0755); err != nil {
		return err
	}

	seen := make(map[string]bool)

	for _, route := range routes {
		if seen[route.Domain] {
			continue
		}
		seen[route.Domain] = true

		// Find target node info to proxy to its IP address
		nodeInfo, err := config.FindNode(route.Node)
		var targetIP string
		if err != nil {
			// default to assuming node name is an IP/hostname if missing in nodes.yaml
			targetIP = route.Node
		} else {
			targetIP = nodeInfo.IP
		}

		vhostContent := fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location / {
        proxy_pass http://%s:%d;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
`, route.Domain, targetIP, route.Port)

		vhostFile := filepath.Join(confDDir, route.Domain+".conf")
		if err := os.WriteFile(vhostFile, []byte(vhostContent), 0644); err != nil {
			return fmt.Errorf("failed to write config for %s: %w", route.Domain, err)
		}
	}

	return nil
}

func getRouterPort() string {
	if p := os.Getenv("PIMAN_ROUTER_PORT"); p != "" {
		return p
	}
	return "80"
}

// StartRouter starts the internal Nginx router
func StartRouter() error {
	if err := RegenerateNginxConfig(); err != nil {
		return err
	}

	// 1. Try Docker container approach
	_, err := exec.LookPath("docker")
	if err == nil {
		// inspect if running
		cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", "piman-router")
		out, inspectErr := cmd.Output()
		if inspectErr == nil && strings.TrimSpace(string(out)) == "true" {
			// Container is already running
			return nil
		}

		// Delete if exists but stopped
		_ = exec.Command("docker", "rm", "-f", "piman-router").Run()

		pDir, err := config.GetPistoreDir()
		if err != nil {
			return err
		}
		routerDir := filepath.Join(pDir, "router")
		nginxConf := filepath.Join(routerDir, "nginx.conf")
		confD := filepath.Join(routerDir, "conf.d")
		port := getRouterPort()

		runCmd := exec.Command("docker", "run", "-d",
			"--name", "piman-router",
			"--restart", "always",
			"-p", port+":80",
			"-v", nginxConf+":/etc/nginx/nginx.conf:ro",
			"-v", confD+":/etc/nginx/conf.d:ro",
			"nginx:alpine",
		)
		if err := runCmd.Run(); err == nil {
			return nil
		}
	}

	// 2. Fallback to host system nginx binary
	nginxBin, err := exec.LookPath("nginx")
	if err != nil {
		return fmt.Errorf("failed to start router: neither docker nor host nginx was found: %w", err)
	}

	pDir, err := config.GetPistoreDir()
	if err != nil {
		return err
	}
	nginxConf := filepath.Join(pDir, "router", "nginx.conf")

	startCmd := exec.Command(nginxBin, "-c", nginxConf)
	return startCmd.Run()
}

// StopRouter stops the Nginx router
func StopRouter() error {
	// 1. Try Docker container approach
	_, err := exec.LookPath("docker")
	if err == nil {
		stopCmd := exec.Command("docker", "rm", "-f", "piman-router")
		return stopCmd.Run()
	}

	// 2. Fallback to host system nginx reload/stop
	nginxBin, err := exec.LookPath("nginx")
	if err == nil {
		pDir, err := config.GetPistoreDir()
		if err != nil {
			return err
		}
		nginxConf := filepath.Join(pDir, "router", "nginx.conf")
		stopCmd := exec.Command(nginxBin, "-c", nginxConf, "-s", "stop")
		return stopCmd.Run()
	}

	return fmt.Errorf("neither docker nor nginx binary found to stop the router")
}

// GetRouterStatus returns the status of the router ("running", "stopped", etc.)
func GetRouterStatus() (string, error) {
	_, err := exec.LookPath("docker")
	if err == nil {
		cmd := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", "piman-router")
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}

	return "stopped", nil
}

// ReloadRouter regenerates configurations and reloads the Nginx service
func ReloadRouter() error {
	if err := RegenerateNginxConfig(); err != nil {
		return err
	}

	_, err := exec.LookPath("docker")
	if err == nil {
		cmd := exec.Command("docker", "exec", "piman-router", "nginx", "-s", "reload")
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	nginxBin, err := exec.LookPath("nginx")
	if err == nil {
		pDir, err := config.GetPistoreDir()
		if err != nil {
			return err
		}
		nginxConf := filepath.Join(pDir, "router", "nginx.conf")
		cmd := exec.Command(nginxBin, "-c", nginxConf, "-s", "reload")
		return cmd.Run()
	}

	return fmt.Errorf("failed to reload router: neither docker nor host nginx was available")
}
