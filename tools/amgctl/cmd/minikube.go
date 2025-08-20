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

Optional tools:
  - minikube (will show warning if missing, required with --single-node)
  - helm (will show warning if missing)

Flags:
  --single-node    Make minikube a required dependency for single-node deployments`,
	RunE: func(cmd *cobra.Command, args []string) error {
		singleNode, _ := cmd.Flags().GetBool("single-node")
		return runMinikubePreFlight(singleNode)
	},
}

func init() {
	minikubeCmd.AddCommand(minikubePreFlightCmd)
	minikubePreFlightCmd.Flags().Bool("single-node", false, "Make minikube a required dependency for single-node deployments")
}

func runMinikubePreFlight(singleNode bool) error {
	fmt.Println("🚀 Minikube Pre-flight Check")
	fmt.Println("==============================")

	// Required tools (base set)
	requiredTools := []string{
		"kubectl",
		"kubeadm",
		"nvidia-smi",
		"nvidia-ctk",
		"docker",
	}

	// Optional tools (base set)
	optionalTools := []string{
		"helm",
	}

	// If single-node mode, minikube becomes required
	if singleNode {
		requiredTools = append(requiredTools, "minikube")
		fmt.Println("Single-node mode: minikube is required")
	} else {
		optionalTools = append(optionalTools, "minikube")
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
