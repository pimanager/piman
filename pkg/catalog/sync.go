package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/sameerchandra/piman/pkg/config"
)

// SyncCatalog clones or pulls the remote Git repository to the local ~/.pistore/catalog_cache directory
func SyncCatalog(repoURL string) (string, error) {
	pistoreDir, err := config.GetPistoreDir()
	if err != nil {
		return "", fmt.Errorf("failed to get pistore directory: %w", err)
	}

	cacheDir := filepath.Join(pistoreDir, "catalog_cache")

	// Check if directory exists
	_, err = os.Stat(cacheDir)
	if os.IsNotExist(err) {
		// Clone repository
		fmt.Printf("Cloning catalog from %s into %s...\n", repoURL, cacheDir)
		_, err = git.PlainClone(cacheDir, false, &git.CloneOptions{
			URL:      repoURL,
			Progress: os.Stdout,
		})
		if err != nil {
			return "", fmt.Errorf("failed to clone catalog: %w", err)
		}
		return cacheDir, nil
	} else if err != nil {
		return "", fmt.Errorf("failed to inspect catalog cache directory: %w", err)
	}

	// Repository exists, open and pull
	fmt.Printf("Opening local catalog cache at %s...\n", cacheDir)
	repo, err := git.PlainOpen(cacheDir)
	if err != nil {
		return "", fmt.Errorf("failed to open local catalog: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get catalog worktree: %w", err)
	}

	fmt.Println("Pulling latest catalog changes...")
	err = wt.Pull(&git.PullOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
	})
	if err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			fmt.Println("Catalog is already up to date.")
			return cacheDir, nil
		}
		return "", fmt.Errorf("failed to pull catalog changes: %w", err)
	}

	fmt.Println("Catalog synchronized successfully.")
	return cacheDir, nil
}
