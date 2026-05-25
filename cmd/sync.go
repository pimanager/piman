package cmd

import (
	"errors"
	"fmt"

	"github.com/sameerchandra/piman/pkg/catalog"
	"github.com/spf13/cobra"
)

var (
	repoURL string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize the application catalog from a Git repository",
	Long:  `Clones or pulls the specified remote catalog Git repository to the local catalog cache directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if repoURL == "" {
			return errors.New("the remote Git repository URL must be specified via the --repo flag")
		}
		path, err := catalog.SyncCatalog(repoURL)
		if err != nil {
			return err
		}
		fmt.Printf("Catalog successfully synchronized at: %s\n", path)
		return nil
	},
}

func init() {
	syncCmd.Flags().StringVarP(&repoURL, "repo", "r", "", "Remote Git repository URL for the app catalog")
	_ = syncCmd.MarkFlagRequired("repo")
	rootCmd.AddCommand(syncCmd)
}
