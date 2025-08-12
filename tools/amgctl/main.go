package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.1"

var rootCmd = &cobra.Command{
	Use:   "amgctl",
	Short: "AMG Scripts Control Tool",
	Long: `amgctl is a command line interface for managing AMG (AI Model Gateway) environments.
It provides tools for setting up, managing, and monitoring AMG environments.`,
	Version: version,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(hostCmd)
	rootCmd.AddCommand(dockerCmd)
	rootCmd.AddCommand(updateCmd)
}
