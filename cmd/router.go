package cmd

import (
	"fmt"

	"github.com/sameerchandra/piman/pkg/router"
	"github.com/spf13/cobra"
)

var routerCmd = &cobra.Command{
	Use:   "router [command]",
	Short: "Manage local inbound Nginx router",
	Long:  `Manages the local Nginx reverse proxy router that maps custom domains to deployed cluster services.`,
}

var routerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Nginx router container",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Starting inbound Nginx router...")
		if err := router.StartRouter(); err != nil {
			return err
		}
		fmt.Println("Router successfully started. Listening on port 80.")
		return nil
	},
}

var routerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Nginx router container",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Stopping inbound Nginx router...")
		if err := router.StopRouter(); err != nil {
			return err
		}
		fmt.Println("Router successfully stopped.")
		return nil
	},
}

var routerReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Regenerate configurations and reload Nginx configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Regenerating configurations and reloading Nginx router...")
		if err := router.ReloadRouter(); err != nil {
			return err
		}
		fmt.Println("Router successfully reloaded.")
		return nil
	},
}

var routerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of Nginx router",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		status, err := router.GetRouterStatus()
		if err != nil {
			return err
		}
		fmt.Printf("Router status: %s\n", status)
		return nil
	},
}

func init() {
	routerCmd.AddCommand(routerStartCmd)
	routerCmd.AddCommand(routerStopCmd)
	routerCmd.AddCommand(routerReloadCmd)
	routerCmd.AddCommand(routerStatusCmd)
	rootCmd.AddCommand(routerCmd)
}
