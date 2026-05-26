package docker

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/sameerchandra/piman/pkg/config"
	"golang.org/x/crypto/ssh"
)

// NewRemoteDockerClient establishes an SSH connection to the node using its private key,
// and returns a Docker API client configured to tunnel its HTTP traffic over the SSH connection
// to the remote Docker Unix socket (/var/run/docker.sock).
// The caller is responsible for closing both the returned client.Client and the ssh.Client.
func NewRemoteDockerClient(nodeName string) (*client.Client, *ssh.Client, error) {
	// 1. Resolve Node info
	node, err := config.FindNode(nodeName)
	if err != nil {
		return nil, nil, err
	}

	// 2. Resolve Private Key path
	privateKeyPath, err := config.GetNodePrivateKeyPath(nodeName)
	if err != nil {
		return nil, nil, err
	}

	// 3. Read and parse private key
	pemBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// 4. Configure SSH connection
	addr := node.IP
	if !strings.Contains(addr, ":") {
		addr = addr + ":22"
	}

	sshConfig := &ssh.ClientConfig{
		User: node.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// 5. Dial SSH
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial SSH to %s: %w", addr, err)
	}

	// 6. Create custom DialContext that routes Unix socket traffic over the SSH tunnel
	dialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return sshClient.Dial("unix", "/var/run/docker.sock")
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: dialer,
		},
	}

	// 7. Create Docker client
	cli, err := client.NewClientWithOpts(
		client.WithHTTPClient(httpClient),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		sshClient.Close()
		return nil, nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return cli, sshClient, nil
}
