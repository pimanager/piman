package cmd

import (
	"errors"
	"os"

	"github.com/sameerchandra/piman/pkg/deploy"
	"github.com/spf13/cobra"
)

var (
	portsList []string
)

var deployCmd = &cobra.Command{
	Use:   "deploy [node] [path-to-docker-compose.yml]",
	Short: "Deploy an application to a worker node",
	Long:  `Deploys an application using a docker-compose.yml file onto a target remote node via Docker Compose over SSH.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]
		composePath := args[1]

		if nodeName == "" {
			return errors.New("node name cannot be empty")
		}
		if composePath == "" {
			return errors.New("compose file path cannot be empty")
		}

		var overrides []deploy.PortOverride
		for _, p := range portsList {
			override, err := deploy.ParsePortOverride(p)
			if err != nil {
				return err
			}
			overrides = append(overrides, override)
		}

		return deploy.Deploy(nodeName, composePath, overrides, os.Stdout)
	},
}

func init() {
	deployCmd.Flags().StringSliceVarP(&portsList, "port", "p", nil, "Customize ports (format: [service:]host_port:container_port)")
	rootCmd.AddCommand(deployCmd)
}
