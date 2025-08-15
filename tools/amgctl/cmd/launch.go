package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch an AMG container",
	Long:  `Launch an AMG container with specified configurations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("amgctl docker launch called")
		return nil
	},
}

func init() {
	dockerCmd.AddCommand(launchCmd)
}
