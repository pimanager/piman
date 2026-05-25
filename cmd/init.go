package cmd

import (
	"fmt"

	"github.com/sameerchandra/piman/pkg/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the local piStore configurations",
	Long: `Initialize the local filesystem configuration for piStore.
Creates ~/.pistore, ~/.pistore/keys/_ed25519, and a template nodes.yaml if they do not exist.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := config.InitPistoreDir()
		if err != nil {
			return err
		}
		fmt.Printf("Successfully initialized piStore configuration directory at: %s\n", dir)
		fmt.Println("Please configure your worker nodes in nodes.yaml and place your SSH private keys in keys/_ed25519.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
