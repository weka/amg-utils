package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var minikubeCmd = &cobra.Command{
	Use:   "minikube",
	Short: "Minikube management commands",
	Long:  `Manage Minikube environments and prerequisites for AMG.`,
}

var minikubePreFlightCmd = &cobra.Command{
	Use:   "pre-flight",
	Short: "Check prerequisites for Minikube AMG deployment",
	Long: `Check whether all required tools are installed and available in PATH.

Required tools:
  - kubectl
  - kubeadm
  - nvidia-smi  
  - nvidia-ctk
  - docker
  - minikube

Optional tools:
  - helm (will show warning if missing)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMinikubePreFlight()
	},
}

func init() {
	minikubeCmd.AddCommand(minikubePreFlightCmd)
}

func runMinikubePreFlight() error {
	fmt.Println("🚀 Minikube Pre-flight Check")
	fmt.Println("==============================")

	// Required tools
	requiredTools := []string{
		"kubectl",
		"kubeadm",
		"nvidia-smi",
		"nvidia-ctk",
		"docker",
		"minikube",
	}

	// Optional tools
	optionalTools := []string{
		"helm",
	}

	var missingRequired []string
	var missingOptional []string

	for _, tool := range requiredTools {
		if !isCommandAvailable(tool) {
			missingRequired = append(missingRequired, tool)
			fmt.Printf("❌ %s: NOT FOUND\n", tool)
		} else {
			fmt.Printf("✅ %s: OK\n", tool)
		}
	}

	for _, tool := range optionalTools {
		if !isCommandAvailable(tool) {
			missingOptional = append(missingOptional, tool)
			fmt.Printf("⚠️  %s: NOT FOUND (optional)\n", tool)
		} else {
			fmt.Printf("✅ %s: OK\n", tool)
		}
	}

	fmt.Println()

	// Report results
	if len(missingRequired) > 0 {
		fmt.Printf("❌ Pre-flight check FAILED. Missing required tools:\n")
		for _, tool := range missingRequired {
			fmt.Printf("   - %s\n", tool)
		}
		fmt.Println("\nPlease install the missing tools and ensure they are available in your PATH.")
		return fmt.Errorf("missing required tools: %v", missingRequired)
	}

	if len(missingOptional) > 0 {
		fmt.Printf("⚠️  Warning: Missing optional tools:\n")
		for _, tool := range missingOptional {
			fmt.Printf("   - %s\n", tool)
		}
		fmt.Println("These tools are optional but recommended for full functionality.")
		fmt.Println()
	}

	fmt.Println("🎉 Pre-flight check PASSED! All required tools are available.")
	return nil
}
