package deploy

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sameerchandra/piman/pkg/config"
	"gopkg.in/yaml.v3"
)

// Deploy runs `docker compose up -d` against the remote worker node using the specified docker-compose file.
// It maps the target node, resolves its SSH key, builds a temporary SSH wrapper script to inject the key,
// and runs the compose command with the DOCKER_HOST environment variable set.
func Deploy(nodeName string, composeFilePath string, overrides []PortOverride, output io.Writer) error {
	// 1. Resolve Node info
	node, err := config.FindNode(nodeName)
	if err != nil {
		return err
	}

	// 2. Resolve Private Key path
	privateKeyPath, err := config.GetNodePrivateKeyPath(nodeName)
	if err != nil {
		return err
	}

	// 3. Locate system ssh binary
	sysSSH, err := exec.LookPath("ssh")
	if err != nil {
		sysSSH = "/usr/bin/ssh" // fallback
	}

	// 4. Create temporary directory for the SSH wrapper
	tmpDir, err := os.MkdirTemp("", "pistore-ssh-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 4.5 Customize compose file if overrides are provided
	if len(overrides) > 0 {
		content, err := os.ReadFile(composeFilePath)
		if err != nil {
			return fmt.Errorf("failed to read compose file %q: %w", composeFilePath, err)
		}
		newContent, err := CustomizeComposePorts(content, overrides)
		if err != nil {
			return fmt.Errorf("failed to customize ports: %w", err)
		}
		customComposePath := filepath.Join(tmpDir, "docker-compose.yml")
		err = os.WriteFile(customComposePath, newContent, 0644)
		if err != nil {
			return fmt.Errorf("failed to write custom compose file: %w", err)
		}
		composeFilePath = customComposePath
	}

	// 5. Write the SSH wrapper script
	wrapperPath := filepath.Join(tmpDir, "ssh")
	wrapperContent := fmt.Sprintf(`#!/bin/sh
exec "%s" -4 -i "%s" -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "$@"
`, sysSSH, privateKeyPath)

	err = os.WriteFile(wrapperPath, []byte(wrapperContent), 0755)
	if err != nil {
		return fmt.Errorf("failed to write SSH wrapper script: %w", err)
	}

	// 6. Formulate DOCKER_HOST
	dockerHost := fmt.Sprintf("ssh://%s@%s", node.Username, node.IP)
	fmt.Printf("Deploying to node %q (%s) using compose file %s...\n", node.Name, dockerHost, composeFilePath)

	// 7. Configure command environment
	cmd := exec.Command("docker", "compose", "-f", composeFilePath, "up", "-d")
	cmd.Stdout = output
	cmd.Stderr = output

	// Modify PATH to prepend the temp directory containing our 'ssh' wrapper
	originalPath := os.Getenv("PATH")
	newPath := tmpDir + string(os.PathListSeparator) + originalPath

	// Set Env
	cmd.Env = append(os.Environ(),
		"DOCKER_HOST="+dockerHost,
		"PATH="+newPath,
	)

	// 8. Execute the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose command failed: %w", err)
	}

	fmt.Printf("Application successfully deployed to node %q.\n", node.Name)
	return nil
}

// PortOverride represents a custom port mapping rule
type PortOverride struct {
	Service       string
	HostPort      string
	ContainerPort string
}

// ParsePortOverride parses a single [service:]host_port:container_port override string
func ParsePortOverride(s string) (PortOverride, error) {
	parts := strings.Split(s, ":")
	var override PortOverride
	if len(parts) == 2 {
		override.HostPort = parts[0]
		override.ContainerPort = parts[1]
	} else if len(parts) == 3 {
		override.Service = parts[0]
		override.HostPort = parts[1]
		override.ContainerPort = parts[2]
	} else {
		return override, fmt.Errorf("invalid port format: %q. Expected [service:]host_port:container_port", s)
	}

	if override.HostPort == "" || override.ContainerPort == "" {
		return override, fmt.Errorf("invalid port format: %q. Host and container ports cannot be empty", s)
	}
	return override, nil
}

// CustomizeComposePorts modifies the compose YAML content based on the provided overrides.
func CustomizeComposePorts(content []byte, overrides []PortOverride) ([]byte, error) {
	if len(overrides) == 0 {
		return content, nil
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	servicesVal, ok := doc["services"]
	if !ok {
		return nil, fmt.Errorf("invalid compose file: 'services' section not found")
	}

	services, ok := servicesVal.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid compose file: 'services' section is not a map")
	}

	// For each override, apply it
	for _, override := range overrides {
		applied := false

		// 1. Try to find existing port mapping and update it
		for sName, sVal := range services {
			if override.Service != "" && sName != override.Service {
				continue
			}

			sMap, ok := sVal.(map[string]interface{})
			if !ok {
				continue
			}

			portsVal, hasPorts := sMap["ports"]
			if !hasPorts {
				continue
			}

			portsList, ok := portsVal.([]interface{})
			if !ok {
				continue
			}

			for idx, portEntry := range portsList {
				matched := false
				var newEntry interface{}

				switch val := portEntry.(type) {
				case int:
					cPortStr := strconv.Itoa(val)
					if cPortStr == override.ContainerPort {
						matched = true
						newEntry = fmt.Sprintf("%s:%s", override.HostPort, override.ContainerPort)
					}
				case string:
					_, container, proto := parsePortString(val)
					if container == override.ContainerPort {
						matched = true
						var hostPart string
						// If original host port was part of an IP mapping (e.g. 127.0.0.1:8080:80)
						parts := strings.Split(val, ":")
						if len(parts) == 3 {
							hostPart = parts[0] + ":" + override.HostPort
						} else {
							hostPart = override.HostPort
						}

						if proto != "" {
							newEntry = fmt.Sprintf("%s:%s/%s", hostPart, override.ContainerPort, proto)
						} else {
							newEntry = fmt.Sprintf("%s:%s", hostPart, override.ContainerPort)
						}
					}
				case map[string]interface{}:
					// Long syntax, e.g. target: 80, published: 8080
					targetVal, targetOk := val["target"]
					if targetOk {
						var targetStr string
						switch t := targetVal.(type) {
						case int:
							targetStr = strconv.Itoa(t)
						case string:
							targetStr = t
						}
						if targetStr == override.ContainerPort {
							matched = true
							if hpInt, err := strconv.Atoi(override.HostPort); err == nil {
								val["published"] = hpInt
							} else {
								val["published"] = override.HostPort
							}
							newEntry = val
						}
					}
				}

				if matched {
					portsList[idx] = newEntry
					applied = true
				}
			}
			sMap["ports"] = portsList
		}

		// 2. If not applied, we need to add the port mapping
		if !applied {
			targetService := override.Service
			if targetService == "" {
				// If no service name was specified, we can only add it if there is exactly one service.
				if len(services) == 1 {
					for sName := range services {
						targetService = sName
					}
				} else {
					return nil, fmt.Errorf("ambiguous port override %q: multiple services found in compose file, please specify service name (e.g., -p service:host:container)", override.HostPort+":"+override.ContainerPort)
				}
			}

			sVal, exists := services[targetService]
			if !exists {
				return nil, fmt.Errorf("service %q not found in compose file", targetService)
			}

			sMap, ok := sVal.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("service %q is not a map", targetService)
			}

			portsVal, hasPorts := sMap["ports"]
			var portsList []interface{}
			if hasPorts {
				portsList, _ = portsVal.([]interface{})
			}

			newEntry := fmt.Sprintf("%s:%s", override.HostPort, override.ContainerPort)
			portsList = append(portsList, newEntry)
			sMap["ports"] = portsList
		}
	}

	// Marshal back to yaml
	newContent, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal modified compose file: %w", err)
	}

	return newContent, nil
}

func parsePortString(s string) (host string, container string, protocol string) {
	if idx := strings.Index(s, "/"); idx != -1 {
		protocol = s[idx+1:]
		s = s[:idx]
	}
	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		container = parts[0]
	case 2:
		host = parts[0]
		container = parts[1]
	case 3:
		host = parts[1]
		container = parts[2]
	}
	return
}
