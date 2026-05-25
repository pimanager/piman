package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "piman",
	Short: "piman is a lightweight, multi-node homeserver orchestrator",
	Long: `piman is a lightweight, multi-node homeserver orchestrator that deploys
services to remote worker Raspberry Pis using Docker Compose over SSH, and
monitors health using the Docker SDK.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Define root command flags here if needed
}
