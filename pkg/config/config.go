package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Node represents a target deployment node (e.g., worker Pi)
type Node struct {
	Name     string `yaml:"name"`
	IP       string `yaml:"ip"`
	Username string `yaml:"username"`
}

// Config represents the schema of nodes.yaml
type Config struct {
	Nodes []Node `yaml:"nodes"`
}

// GetPistoreDir returns the path to ~/.pistore
func GetPistoreDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pistore"), nil
}

// InitPistoreDir initializes ~/.pistore, creating nodes.yaml and the keys directory if they do not exist
func InitPistoreDir() (string, error) {
	dir, err := GetPistoreDir()
	if err != nil {
		return "", err
	}

	// Create ~/.pistore base directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Create ~/.pistore/keys/_ed25519 directory (stores SSH private keys)
	keysDir := filepath.Join(dir, "keys", "_ed25519")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return "", err
	}

	// Create ~/.pistore/nodes.yaml if it does not exist
	nodesPath := filepath.Join(dir, "nodes.yaml")
	if _, err := os.Stat(nodesPath); os.IsNotExist(err) {
		defaultConfig := Config{
			Nodes: []Node{
				{
					Name:     "example-worker",
					IP:       "192.168.1.100",
					Username: "pi",
				},
			},
		}

		data, err := yaml.Marshal(&defaultConfig)
		if err != nil {
			return "", err
		}

		if err := os.WriteFile(nodesPath, data, 0644); err != nil {
			return "", err
		}
	}

	return dir, nil
}

// LoadConfig loads the nodes.yaml configuration file
func LoadConfig() (*Config, error) {
	dir, err := GetPistoreDir()
	if err != nil {
		return nil, err
	}

	nodesPath := filepath.Join(dir, "nodes.yaml")
	data, err := os.ReadFile(nodesPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// FindNode searches nodes.yaml for a node with the specified name
func FindNode(name string) (*Node, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	for _, node := range cfg.Nodes {
		if node.Name == name {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("node %q not found in nodes.yaml config", name)
}

// GetNodePrivateKeyPath locates the SSH private key for a given node.
// It checks ~/.pistore/keys/_ed25519/<nodeName>. If that is missing, it falls back to
// ~/.pistore/keys/_ed25519/id_ed25519. If neither exists, it returns an error.
func GetNodePrivateKeyPath(nodeName string) (string, error) {
	dir, err := GetPistoreDir()
	if err != nil {
		return "", err
	}

	keysDir := filepath.Join(dir, "keys", "_ed25519")

	// 1. Check node-specific key
	nodeKeyPath := filepath.Join(keysDir, nodeName)
	if _, err := os.Stat(nodeKeyPath); err == nil {
		return nodeKeyPath, nil
	}

	// 2. Check fallback generic key
	fallbackKeyPath := filepath.Join(keysDir, "id_ed25519")
	if _, err := os.Stat(fallbackKeyPath); err == nil {
		return fallbackKeyPath, nil
	}

	return "", fmt.Errorf("no SSH private key found for node %q (checked %s and %s)", nodeName, nodeKeyPath, fallbackKeyPath)
}

// SaveConfig writes the nodes.yaml configuration file
func SaveConfig(cfg *Config) error {
	dir, err := GetPistoreDir()
	if err != nil {
		return err
	}

	nodesPath := filepath.Join(dir, "nodes.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(nodesPath, data, 0644)
}

// AddNode adds a new node to the configuration
func AddNode(node Node) error {
	if node.Name == "" || node.IP == "" || node.Username == "" {
		return fmt.Errorf("node fields (name, ip, username) cannot be empty")
	}

	cfg, err := LoadConfig()
	if err != nil {
		// If nodes.yaml doesn't exist, initialize base first
		_, err = InitPistoreDir()
		if err != nil {
			return err
		}
		cfg, err = LoadConfig()
		if err != nil {
			return err
		}
	}

	// Check for duplicates
	for _, n := range cfg.Nodes {
		if n.Name == node.Name {
			return fmt.Errorf("node with name %q already exists", node.Name)
		}
	}

	cfg.Nodes = append(cfg.Nodes, node)
	return SaveConfig(cfg)
}


