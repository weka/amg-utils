package cmd

import (
	"fmt"
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
	Long: `Check whether all required tools are installed and verify Kubernetes cluster health.

Required tools:
  - kubectl
  - kubeadm
  - nvidia-smi  
  - nvidia-ctk
  - docker
  - helm

Cluster health checks:
  - API server connectivity
  - Node readiness status
  - kube-system pods health
  - Deployment permissions`,
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
		"helm",
	}

	// Optional tools
	optionalTools := []string{}

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
	if err := checkNvidiaPeermemModule(); err != nil {
		return fmt.Errorf("nvidia_peermem module check failed: %w", err)
	}

	// Check Kubernetes cluster health
	fmt.Println("--- Kubernetes Cluster Checks ---")
	if err := checkKubernetesClusterHealth(); err != nil {
		return fmt.Errorf("kubernetes cluster health check failed: %w", err)
	}

	fmt.Println("🎉 Pre-flight check PASSED! All required tools are available and Kubernetes cluster is ready.")
	return nil
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// checkKubernetesClusterHealth performs minimal checks to verify Kubernetes cluster is ready for helm deployments
func checkKubernetesClusterHealth() error {
	// Check 1: Basic cluster connectivity and API server health
	fmt.Print("Checking cluster connectivity... ")
	cmd := exec.Command("kubectl", "cluster-info", "--request-timeout=10s")
	if err := cmd.Run(); err != nil {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("cluster-info failed - Kubernetes API server may not be accessible: %w", err)
	}
	fmt.Println("✅ OK")

	// Check 2: Verify nodes are ready
	fmt.Print("Checking node readiness... ")
	cmd = exec.Command("kubectl", "get", "nodes", "--no-headers", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.conditions[?(@.type=='Ready')].status")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("failed to get node status: %w", err)
	}

	// Parse output to check if all nodes are Ready
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("no nodes found in cluster")
	}

	var notReadyNodes []string
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			nodeName := parts[0]
			status := parts[1]
			if status != "True" {
				notReadyNodes = append(notReadyNodes, nodeName)
			}
		}
	}

	if len(notReadyNodes) > 0 {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("nodes not ready: %v", notReadyNodes)
	}
	fmt.Printf("✅ OK (%d nodes ready)\n", len(lines))

	// Check 3: Verify kube-system namespace is healthy (essential pods are running)
	fmt.Print("Checking kube-system health... ")
	cmd = exec.Command("kubectl", "get", "pods", "-n", "kube-system", "--no-headers", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase")
	output, err = cmd.Output()
	if err != nil {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("failed to get kube-system pods: %w", err)
	}

	lines = strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("no pods found in kube-system namespace")
	}

	var nonRunningPods []string
	runningPods := 0
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			podName := parts[0]
			status := parts[1]
			if status == "Running" || status == "Succeeded" {
				runningPods++
			} else {
				nonRunningPods = append(nonRunningPods, fmt.Sprintf("%s(%s)", podName, status))
			}
		}
	}

	// Allow some pods to be non-running, but require at least core components to be healthy
	if runningPods == 0 {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("no running pods in kube-system namespace")
	}

	if len(nonRunningPods) > 0 {
		fmt.Printf("⚠️  WARNING - Some kube-system pods not running: %v\n", nonRunningPods)
	} else {
		fmt.Printf("✅ OK (%d pods running)\n", runningPods)
	}

	// Check 4: Verify we can create/list resources (basic RBAC check)
	fmt.Print("Checking cluster permissions... ")
	cmd = exec.Command("kubectl", "auth", "can-i", "create", "deployments", "--quiet")
	if err := cmd.Run(); err != nil {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("insufficient permissions to create deployments - may not be able to install helm charts")
	}
	fmt.Println("✅ OK")

	fmt.Println("✅ Kubernetes cluster is healthy and ready for helm deployments")
	return nil
}
