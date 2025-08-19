package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const version = "0.1.11"

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "amgctl",
		Short: "AMG Control CLI",
		Long: `amgctl is a command line interface for managing Weka AMG (Augmented Memory Grid) environments.
It provides tools for setting up, managing, and monitoring AMG environments.`,
		Version:    version,
		SuggestFor: []string{"status"}, // Suggest "amgctl" when user types "amgctl status"
	}
)

func Execute() {
	// Configure Cobra to provide better error handling
	rootCmd.SilenceErrors = true           // Prevent duplicate error messages
	rootCmd.SuggestionsMinimumDistance = 1 // More sensitive to typos

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		// Provide specific guidance for common mistakes
		if err.Error() == `unknown command "status" for "amgctl"` {
			fmt.Fprintf(os.Stderr, "\nDid you mean:\n  amgctl host status\n")
		}
		if err.Error() == `unknown command "launch" for "amgctl"` {
			fmt.Fprintf(os.Stderr, "\nDid you mean one of:\n  amgctl docker launch <model>\n  amgctl host launch <model>\n")
		}
		if err.Error() == `unknown command "pre-flight" for "amgctl"` {
			fmt.Fprintf(os.Stderr, "\nDid you mean one of:\n  amgctl host pre-flight\n  amgctl k8s pre-flight\n")
		}

		fmt.Fprintf(os.Stderr, "\nRun 'amgctl --help' for usage.\n")
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/amgctl.yaml)")
	// Add subcommands
	rootCmd.AddCommand(dockerCmd)
	rootCmd.AddCommand(hostCmd)
	rootCmd.AddCommand(hwCmd)
	rootCmd.AddCommand(k8sCmd)
	rootCmd.AddCommand(updateCmd)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".config/amgctl" (without extension).
		viper.AddConfigPath(fmt.Sprintf("%s/.config", home))
		viper.SetConfigName("amgctl")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("AMGCTL")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
