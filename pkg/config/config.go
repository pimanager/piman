package config

import (
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
