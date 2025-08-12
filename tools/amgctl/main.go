package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.4"

var rootCmd = &cobra.Command{
	Use:   "amgctl",
	Short: "AMG Control CLI",
	Long: `amgctl is a command line interface for managing Weka AMG (Augmented Memory Grid) environments.
It provides tools for setting up, managing, and monitoring AMG environments.`,
	Version:    version,
	SuggestFor: []string{"status"}, // Suggest "amgctl" when user types "amgctl status"
}

func main() {
	// Configure Cobra to provide better error handling
	rootCmd.SilenceErrors = true           // Prevent duplicate error messages
	rootCmd.SuggestionsMinimumDistance = 1 // More sensitive to typos

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		// Provide specific guidance for common mistakes
		if err.Error() == `unknown command "status" for "amgctl"` {
			fmt.Fprintf(os.Stderr, "\nDid you mean:\n  amgctl host status\n")
		}

		fmt.Fprintf(os.Stderr, "\nRun 'amgctl --help' for usage.\n")
		os.Exit(1)
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(hostCmd)
	rootCmd.AddCommand(dockerCmd)
	rootCmd.AddCommand(updateCmd)
}
