package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/sameerchandra/piman/pkg/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap [node-name]",
	Short: "Authorize SSH key on a remote node using password",
	Long: `Connects to the target node using password authentication,
and installs the node's public key into the remote ~/.ssh/authorized_keys file.
If the SSH key does not exist locally, it will be automatically generated.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]
		if nodeName == "" {
			return errors.New("node-name cannot be empty")
		}

		fmt.Print("Enter remote SSH password: ")
		bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println() // Print newline after hidden input

		password := string(bytePassword)
		if password == "" {
			return errors.New("password cannot be empty")
		}

		return config.BootstrapNode(nodeName, password)
	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}
