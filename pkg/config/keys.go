package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// GenerateNodeKey generates an Ed25519 SSH keypair for a node.
// It writes the private key to ~/.pistore/keys/_ed25519/<nodeName> (0600)
// and the public key to ~/.pistore/keys/_ed25519/<nodeName>.pub (0644).
func GenerateNodeKey(nodeName string) (string, string, error) {
	dir, err := GetPistoreDir()
	if err != nil {
		return "", "", err
	}

	keysDir := filepath.Join(dir, "keys", "_ed25519")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return "", "", err
	}

	privPath := filepath.Join(keysDir, nodeName)
	pubPath := privPath + ".pub"

	// Generate keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Marshal private key to OpenSSH PEM block
	pemBlock, err := ssh.MarshalPrivateKey(priv, "piStore generated key")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	privBytes := pem.EncodeToMemory(pemBlock)
	if err := os.WriteFile(privPath, privBytes, 0600); err != nil {
		return "", "", fmt.Errorf("failed to write private key: %w", err)
	}

	// Generate public key in SSH authorized_keys format
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", "", fmt.Errorf("failed to create SSH public key: %w", err)
	}

	pubBytes := ssh.MarshalAuthorizedKey(sshPub)
	if err := os.WriteFile(pubPath, pubBytes, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write public key: %w", err)
	}

	return privPath, string(pubBytes), nil
}
