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

var k8sDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy AMG to Kubernetes cluster",
	Long: `Deploy AMG to Kubernetes cluster after running pre-flight checks.

This command will:
1. Run pre-flight checks to verify cluster readiness
2. Add required helm repositories (nvidia)
3. Update helm repositories
4. Install the AMG chart with optimal settings

The deployment includes:
- NVIDIA helm repository for dependencies
- AMG chart from OCI registry (ghcr.io/sdimitro/amg-chart)
- 30-minute timeout with wait and debug flags for monitoring

Examples:
  amgctl k8s deploy`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runK8sDeploy()
	},
}

var k8sRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove AMG deployment from Kubernetes cluster",
	Long: `Remove AMG deployment from Kubernetes cluster with verification.

This command will:
1. Check if AMG release exists
2. Uninstall the AMG helm release with configurable timeout
3. Verify removal was successful
4. Optionally remove helm repositories

The removal process includes:
- Safe uninstallation of amg-release helm chart with 20-minute default timeout
- Verification that all AMG resources are cleaned up
- Optional cleanup of NVIDIA helm repository

Examples:
  amgctl k8s remove
  amgctl k8s remove --remove-repos
  amgctl k8s remove --timeout 30m
  amgctl k8s remove --timeout 1h --remove-repos`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runK8sRemove(cmd)
	},
}

func init() {
	k8sCmd.AddCommand(k8sPreFlightCmd)
	k8sCmd.AddCommand(k8sDeployCmd)
	k8sCmd.AddCommand(k8sRemoveCmd)

	// Add flags for remove command
	k8sRemoveCmd.PersistentFlags().Bool("remove-repos", false, "Also remove helm repositories (nvidia) that were added during deployment")
	k8sRemoveCmd.PersistentFlags().String("timeout", "20m", "Timeout duration for helm uninstall operation (e.g., 10m, 30m, 1h)")
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

// runK8sDeploy performs pre-flight checks and then deploys AMG to the Kubernetes cluster
func runK8sDeploy() error {
	fmt.Println("🚀 AMG Kubernetes Deployment")
	fmt.Println("============================")
	fmt.Println()

	// Step 1: Run pre-flight checks first
	fmt.Println("📋 Step 1: Running pre-flight checks...")
	if err := runK8sPreFlight(); err != nil {
		return fmt.Errorf("pre-flight checks failed: %w", err)
	}
	fmt.Println()

	// Step 2: Add required helm repositories
	fmt.Println("📦 Step 2: Setting up helm repositories...")
	fmt.Print("Adding NVIDIA helm repository... ")
	cmd := exec.Command("helm", "repo", "add", "nvidia", "https://helm.ngc.nvidia.com/nvidia")
	if err := cmd.Run(); err != nil {
		// Check if repo already exists (this is OK)
		if strings.Contains(err.Error(), "already exists") {
			fmt.Println("✅ Already exists")
		} else {
			fmt.Println("❌ FAILED")
			return fmt.Errorf("failed to add nvidia helm repository: %w", err)
		}
	} else {
		fmt.Println("✅ OK")
	}

	fmt.Print("Updating helm repositories... ")
	cmd = exec.Command("helm", "repo", "update")
	if err := cmd.Run(); err != nil {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("failed to update helm repositories: %w", err)
	}
	fmt.Println("✅ OK")
	fmt.Println()

	// Step 3: Install the AMG chart
	fmt.Println("🎯 Step 3: Installing AMG chart...")
	fmt.Println("This may take up to 30 minutes depending on cluster resources...")
	fmt.Print("Installing AMG release... ")

	cmd = exec.Command("helm", "install", "amg-release",
		"oci://ghcr.io/sdimitro/amg-chart",
		"--version", "0.1.0",
		"--wait",
		"--timeout=30m",
		"--debug")

	// Stream output in real-time for better user experience
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("failed to install AMG chart: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ AMG chart installed successfully!")
	fmt.Println()

	// Step 4: Verify deployment
	fmt.Println("🔍 Step 4: Verifying deployment...")
	fmt.Print("Checking AMG release status... ")
	cmd = exec.Command("helm", "status", "amg-release")
	if err := cmd.Run(); err != nil {
		fmt.Println("⚠️  WARNING - Could not verify release status")
	} else {
		fmt.Println("✅ OK")
	}

	fmt.Print("Checking AMG pods... ")
	cmd = exec.Command("kubectl", "get", "pods", "-l", "app.kubernetes.io/instance=amg-release", "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("⚠️  WARNING - Could not check pod status")
	} else {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 && lines[0] != "" {
			fmt.Printf("✅ OK (%d pods found)\n", len(lines))
		} else {
			fmt.Println("⚠️  WARNING - No AMG pods found")
		}
	}

	fmt.Println()
	fmt.Println("🎉 AMG deployment completed successfully!")
	fmt.Println()
	fmt.Println("📋 Next Steps:")
	fmt.Println("  • Check deployment status: helm status amg-release")
	fmt.Println("  • View AMG pods: kubectl get pods -l app.kubernetes.io/instance=amg-release")
	fmt.Println("  • View AMG services: kubectl get services -l app.kubernetes.io/instance=amg-release")
	fmt.Println("  • View logs: kubectl logs -l app.kubernetes.io/instance=amg-release")

	return nil
}

// runK8sRemove removes the AMG deployment from the Kubernetes cluster
func runK8sRemove(cmd *cobra.Command) error {
	fmt.Println("🗑️  AMG Kubernetes Removal")
	fmt.Println("=========================")
	fmt.Println()

	// Step 1: Check if AMG release exists
	fmt.Println("🔍 Step 1: Checking AMG deployment...")
	fmt.Print("Checking for amg-release... ")
	execCmd := exec.Command("helm", "status", "amg-release", "--output", "json")
	output, err := execCmd.Output()
	if err != nil {
		fmt.Println("❌ NOT FOUND")
		fmt.Println()
		fmt.Println("ℹ️  No AMG release found. Nothing to remove.")
		fmt.Println("💡 If you're looking for a different release name, use: helm list")
		return nil
	}
	fmt.Println("✅ FOUND")

	// Parse basic info about the release
	if len(output) > 0 {
		fmt.Println("📋 Release details:")
		// Get basic status without full JSON parsing
		statusCmd := exec.Command("helm", "status", "amg-release", "--output", "table")
		if statusOutput, err := statusCmd.Output(); err == nil {
			// Show just the first few lines for context
			lines := strings.Split(string(statusOutput), "\n")
			for i, line := range lines {
				if i < 5 && strings.TrimSpace(line) != "" { // Show first 5 non-empty lines
					fmt.Printf("  %s\n", line)
				}
			}
		}
	}
	fmt.Println()

	// Step 2: Get user confirmation (implicit through command execution)
	fmt.Println("⚠️  This will remove the AMG deployment and all associated resources.")
	fmt.Println()

	// Step 3: Remove the AMG helm release
	timeout, _ := cmd.Flags().GetString("timeout")
	fmt.Println("🗑️  Step 2: Removing AMG deployment...")
	fmt.Printf("Uninstalling amg-release (timeout: %s)... ", timeout)

	execCmd = exec.Command("helm", "uninstall", "amg-release", "--wait", "--timeout="+timeout)

	// Stream output for better user experience
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("failed to uninstall AMG release: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ AMG release uninstalled successfully!")
	fmt.Println()

	// Step 4: Verify removal
	fmt.Println("🔍 Step 3: Verifying removal...")
	fmt.Print("Checking for remaining AMG resources... ")

	// Check for any remaining pods
	execCmd = exec.Command("kubectl", "get", "pods", "-l", "app.kubernetes.io/instance=amg-release", "--no-headers")
	output, err = execCmd.Output()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		fmt.Println("⚠️  WARNING - Some pods still exist")
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("    %s\n", line)
			}
		}
	} else {
		fmt.Println("✅ OK - No AMG pods found")
	}

	// Check for any remaining services
	fmt.Print("Checking for remaining AMG services... ")
	execCmd = exec.Command("kubectl", "get", "services", "-l", "app.kubernetes.io/instance=amg-release", "--no-headers")
	output, err = execCmd.Output()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		fmt.Println("⚠️  WARNING - Some services still exist")
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("    %s\n", line)
			}
		}
	} else {
		fmt.Println("✅ OK - No AMG services found")
	}

	// Step 5: Optionally remove helm repositories
	removeRepos, _ := cmd.Flags().GetBool("remove-repos")
	if removeRepos {
		fmt.Println()
		fmt.Println("📦 Step 4: Removing helm repositories...")
		fmt.Print("Removing nvidia helm repository... ")
		execCmd = exec.Command("helm", "repo", "remove", "nvidia")
		if err := execCmd.Run(); err != nil {
			// Check if repo doesn't exist (this is OK)
			if strings.Contains(err.Error(), "no repo named") {
				fmt.Println("ℹ️  Already removed")
			} else {
				fmt.Println("⚠️  WARNING - Could not remove nvidia repository")
				fmt.Printf("    Error: %v\n", err)
			}
		} else {
			fmt.Println("✅ OK")
		}
	}

	fmt.Println()
	fmt.Println("🎉 AMG removal completed successfully!")
	fmt.Println()
	fmt.Println("📋 Summary:")
	fmt.Println("  • AMG helm release 'amg-release' has been uninstalled")
	fmt.Println("  • Associated Kubernetes resources have been cleaned up")
	if removeRepos {
		fmt.Println("  • NVIDIA helm repository has been removed")
	} else {
		fmt.Println("  • NVIDIA helm repository was preserved (use --remove-repos to remove)")
	}
	fmt.Println()
	fmt.Println("💡 To verify complete removal:")
	fmt.Println("  • Check helm releases: helm list")
	fmt.Println("  • Check for remaining resources: kubectl get all -l app.kubernetes.io/instance=amg-release")

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
