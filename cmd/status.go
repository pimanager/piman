package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sameerchandra/piman/pkg/docker"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [node-name]",
	Short: "Check application health on a remote node",
	Long:  `Queries the remote Docker daemon on the node and reports the health of all deployed applications.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]
		if nodeName == "" {
			return errors.New("node-name cannot be empty")
		}

		appStatuses, err := docker.GetNodeStatus(nodeName)
		if err != nil {
			return err
		}

		fmt.Printf("piStore Application Status for Node: %s\n", nodeName)
		fmt.Println(strings.Repeat("=", 70))
		fmt.Printf("%-24s %-12s %-32s\n", "APPLICATION", "HEALTH", "SERVICES")
		fmt.Println(strings.Repeat("-", 70))

		for _, app := range appStatuses {
			var svcs []string
			for _, svc := range app.Services {
				svcs = append(svcs, fmt.Sprintf("%s (%s)", svc.Name, svc.Status))
			}
			servicesStr := strings.Join(svcs, ", ")
			if servicesStr == "" {
				servicesStr = "no services defined"
			}
			fmt.Printf("%-24s %-12s %-32s\n", app.AppName, app.Health, servicesStr)
		}
		fmt.Println(strings.Repeat("=", 70))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
