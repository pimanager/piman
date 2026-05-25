package config

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// BootstrapNode connects to the target node using password authentication,
// and appends the node's public SSH key to the remote ~/.ssh/authorized_keys file.
func BootstrapNode(nodeName string, password string) error {
	// 1. Find Node
	node, err := FindNode(nodeName)
	if err != nil {
		return err
	}

	// 2. Ensure Key exists (generate if missing)
	privPath, err := GetNodePrivateKeyPath(nodeName)
	if err != nil {
		// Key is missing, generate it
		fmt.Printf("SSH key pair not found for node %q. Generating new key pair...\n", nodeName)
		privPath, _, err = GenerateNodeKey(nodeName)
		if err != nil {
			return fmt.Errorf("failed to generate key pair: %w", err)
		}
	}

	pubPath := privPath + ".pub"
	pubBytes, err := os.ReadFile(pubPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}
	pubKeyStr := strings.TrimSpace(string(pubBytes))

	// 3. Connect via SSH with password
	addr := node.IP
	if !strings.Contains(addr, ":") {
		addr = addr + ":22"
	}

	config := &ssh.ClientConfig{
		User: node.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	fmt.Printf("Connecting to %s@%s using password...\n", node.Username, addr)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to remote host: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// 4. Install public key remotely
	// We check if the key already exists before appending to keep it idempotent.
	cmd := fmt.Sprintf(`mkdir -p ~/.ssh && chmod 700 ~/.ssh && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && (grep -qF "%s" ~/.ssh/authorized_keys || echo "%s" >> ~/.ssh/authorized_keys)`, pubKeyStr, pubKeyStr)

	fmt.Println("Installing SSH public key on remote host...")
	err = session.Run(cmd)
	if err != nil {
		return fmt.Errorf("failed to execute command on remote host: %w", err)
	}

	fmt.Printf("Successfully bootstrapped node %q! SSH key is now authorized.\n", nodeName)
	return nil
}
