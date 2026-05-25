package cmd

import (
	"errors"
	"fmt"

	"github.com/sameerchandra/piman/pkg/config"
	"github.com/spf13/cobra"
)

var keygenCmd = &cobra.Command{
	Use:   "keygen [node-name]",
	Short: "Generate SSH key pair for a worker node",
	Long: `Generates a new Ed25519 SSH private/public key pair for a worker node.
The private key is saved securely in ~/.pistore/keys/_ed25519/<node-name>
and the public key is saved in ~/.pistore/keys/_ed25519/<node-name>.pub.
Prints out the public key to add to the node's ~/.ssh/authorized_keys file.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]
		if nodeName == "" {
			return errors.New("node-name cannot be empty")
		}

		privPath, pubKeyContent, err := config.GenerateNodeKey(nodeName)
		if err != nil {
			return err
		}

		fmt.Printf("Successfully generated key pair for node: %s\n", nodeName)
		fmt.Printf("Private key saved to: %s\n\n", privPath)
		fmt.Println("--- SSH PUBLIC KEY (authorized_keys format) ---")
		fmt.Print(pubKeyContent)
		fmt.Println("-----------------------------------------------")
		fmt.Println("Please copy the public key above and append it to the file ~/.ssh/authorized_keys on the worker Pi.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(keygenCmd)
}
