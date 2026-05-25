package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// Repository exists, open it
	fmt.Printf("Opening local catalog cache at %s...\n", cacheDir)
	repo, err := git.PlainOpen(cacheDir)
	if err != nil {
		// If opening fails (e.g. cache is corrupted), clean and re-clone
		fmt.Printf("Failed to open local catalog cache, recreating: %v\n", err)
		if err := os.RemoveAll(cacheDir); err != nil {
			return "", fmt.Errorf("failed to clean up corrupted catalog cache: %w", err)
		}
		fmt.Printf("Cloning catalog from %s into %s...\n", repoURL, cacheDir)
		_, err = git.PlainClone(cacheDir, false, &git.CloneOptions{
			URL:      repoURL,
			Progress: os.Stdout,
		})
		if err != nil {
			return "", fmt.Errorf("failed to clone catalog: %w", err)
		}
		return cacheDir, nil
	}

	// Verify that the remote URL matches the requested repoURL
	hasMatchingRemote := false
	remote, err := repo.Remote("origin")
	if err == nil && remote != nil {
		for _, url := range remote.Config().URLs {
			if isSameGitURL(url, repoURL) {
				hasMatchingRemote = true
				break
			}
		}
	}

	if !hasMatchingRemote {
		fmt.Printf("Cached catalog remote URL does not match %s. Re-cloning...\n", repoURL)
		if err := os.RemoveAll(cacheDir); err != nil {
			return "", fmt.Errorf("failed to clean up outdated catalog cache: %w", err)
		}
		fmt.Printf("Cloning catalog from %s into %s...\n", repoURL, cacheDir)
		_, err = git.PlainClone(cacheDir, false, &git.CloneOptions{
			URL:      repoURL,
			Progress: os.Stdout,
		})
		if err != nil {
			return "", fmt.Errorf("failed to clone catalog: %w", err)
		}
		return cacheDir, nil
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

func isSameGitURL(url1, url2 string) bool {
	clean := func(u string) string {
		u = strings.TrimSpace(u)
		u = strings.TrimSuffix(u, "/")
		u = strings.TrimSuffix(u, ".git")
		return strings.ToLower(u)
	}
	return clean(url1) == clean(url2)
}

