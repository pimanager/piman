package cmd

import (
	"fmt"

	"github.com/sameerchandra/piman/pkg/api"
	"github.com/spf13/cobra"
)

var (
	port int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the REST API server",
	Long:  `Starts a lightweight REST API server to expose piStore operations via HTTP.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := fmt.Sprintf(":%d", port)
		return api.StartServer(addr)
	},
}

func init() {
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
