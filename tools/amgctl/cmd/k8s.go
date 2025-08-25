package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Kubernetes management commands",
	Long:  `Manage Kubernetes environments and prerequisites for AMG.`,
}

var k8sPreFlightCmd = &cobra.Command{
	Use:   "pre-flight",
	Short: "Check prerequisites for Kubernetes AMG deployment",
	Long: `Check whether all required tools are installed and available in PATH.

Required tools:
  - kubectl
  - kubeadm
  - nvidia-smi  
  - nvidia-ctk
  - docker

Optional tools:
  - helm (will show warning if missing)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runK8sPreFlight()
	},
}

func init() {
	k8sCmd.AddCommand(k8sPreFlightCmd)
}

func runK8sPreFlight() error {
	fmt.Println("🚀 Kubernetes Pre-flight Check")
	fmt.Println("==============================")

	// Required tools
	requiredTools := []string{
		"kubectl",
		"kubeadm",
		"nvidia-smi",
		"nvidia-ctk",
		"docker",
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

	// Check nvidia_peermem kernel module
	fmt.Println("--- Kernel Module Checks ---")
	if err := checkK8sNvidiaPeermemModule(); err != nil {
		return fmt.Errorf("nvidia_peermem module check failed: %w", err)
	}

	fmt.Println("🎉 Pre-flight check PASSED! All required tools are available.")
	return nil
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func checkK8sNvidiaPeermemModule() error {
	moduleName := "nvidia_peermem"

	if err := checkK8sKernelModuleLoaded(moduleName); err == nil {
		fmt.Println("✅ nvidia_peermem module is loaded")
		return nil
	}

	if err := checkK8sKernelModuleExists(moduleName); err != nil {
		return fmt.Errorf("nvidia_peermem module not found. Please install the nvidia_peermem module")
	}

	return fmt.Errorf("nvidia_peermem module found but not loaded. Please load it with: sudo modprobe %s", moduleName)
}

func checkK8sKernelModuleExists(moduleName string) error {
	cmd := exec.Command("modinfo", moduleName)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("module not found")
	}

	if len(output) == 0 {
		return fmt.Errorf("module exists but modinfo returned no information")
	}

	return nil
}

func checkK8sKernelModuleLoaded(moduleName string) error {
	data, err := os.ReadFile("/proc/modules")
	if err != nil {
		return fmt.Errorf("failed to read /proc/modules: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, moduleName+" ") || strings.HasPrefix(line, moduleName+"\t") {
			return nil
		}
	}

	return fmt.Errorf("module not loaded")
}
