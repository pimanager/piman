package cmd

import (
	"fmt"
	"os/exec"

	"github.com/sameerchandra/piman/pkg/api"
	"github.com/spf13/cobra"
)

var (
	port int
)

func verifyRouterPrerequisites() error {
	_, errDocker := exec.LookPath("docker")
	_, errNginx := exec.LookPath("nginx")
	if errDocker != nil && errNginx != nil {
		return fmt.Errorf("prerequisite check failed: neither Docker nor Nginx is installed on this system.\npiMan requires at least Docker or Nginx to manage router operations.\nPlease install Docker or Nginx and try again")
	}
	return nil
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the REST API server",
	Long:  `Starts a lightweight REST API server to expose piStore operations via HTTP.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := verifyRouterPrerequisites(); err != nil {
			return err
		}
		addr := fmt.Sprintf(":%d", port)
		return api.StartServer(addr)
	},
}

func init() {
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
