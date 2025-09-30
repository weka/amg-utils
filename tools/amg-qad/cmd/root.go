package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const version = "0.1.19"

// GetVersion returns the current application version
func GetVersion() string {
	return version
}

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "amg-qad",
		Short: "AMG Quality Assurance Daemon",
		Long: `amg-qad is a daemon that runs scheduled QA tests for AMG environments.
It provides automated testing, result storage, and a web dashboard to view test results.`,
		Version: version,
	}
)

func Execute() {
	rootCmd.SilenceErrors = true
	rootCmd.SuggestionsMinimumDistance = 1

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nRun 'amg-qad --help' for usage.\n")
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/amg-qad.yaml)")

	// Add subcommands
	rootCmd.AddCommand(daemonCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(fmt.Sprintf("%s/.config", home))
		viper.AddConfigPath(".")
		viper.SetConfigName("amg-qad")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("AMG_QAD")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}
}
