package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sameerchandra/piman/pkg/config"
)

// Deploy runs `docker compose up -d` against the remote worker node using the specified docker-compose file.
// It maps the target node, resolves its SSH key, builds a temporary SSH wrapper script to inject the key,
// and runs the compose command with the DOCKER_HOST environment variable set.
func Deploy(nodeName string, composeFilePath string) error {
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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
