package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"strings"
	"time"

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
4. Install the AMG chart in dedicated 'amg' namespace

The deployment includes:
- NVIDIA helm repository for dependencies
- AMG chart from OCI registry (ghcr.io/sdimitro/amg-chart)
- Installation in isolated 'amg' namespace (auto-created)
- Configurable timeout (default: 30 minutes) with wait and debug flags
- Optional RDMA configuration for hardware-specific setups

RDMA Configuration:
When RDMA flags are provided, the command automatically:
- Enables NicClusterPolicy in the chart
- Configures RDMA shared device plugin with specified parameters
- Passes configuration directly to helm using --set flags

Examples:
  amgctl k8s deploy                                    # Deploy with default settings
  amgctl k8s deploy --timeout 1h                      # Deploy with custom timeout
  amgctl k8s deploy --gpus 4 --ib-devs 2              # Deploy with 4 GPUs and 2 IB devices per pod
  amgctl k8s deploy --rdma-ifNames ibp24s0,ibp206s0   # Deploy with RDMA interface names
  amgctl k8s deploy --rdma-ifNames ibp24s0,ibp206s0,ibp220s0,ibp64s0 \
                    --rdma-vendors 15b3               # Deploy with interfaces and vendor filter
  amgctl k8s deploy --gpus 2 --ib-devs 2 \
                    --rdma-ifNames ibp24s0,ibp206s0 \
                    --timeout 45m                      # Deploy with custom resource and RDMA config`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runK8sDeploy(cmd)
	},
}

var k8sRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove AMG deployment from Kubernetes cluster",
	Long: `Remove AMG deployment from Kubernetes cluster with verification.

This command will:
1. Check if AMG release exists in 'amg' namespace
2. Uninstall the AMG helm release with configurable timeout
3. Wait for namespace cleanup and verify removal
4. Delete the 'amg' namespace by default for complete cleanup
5. Optionally remove helm repositories

The removal process includes:
- Safe uninstallation of amg-release helm chart from 'amg' namespace
- 20-minute default timeout for helm uninstall operation
- Verification that all AMG resources are cleaned up
- Automatic wait for namespace resource cleanup (up to 30 seconds)
- Automatic deletion of the 'amg' namespace for complete cleanup
- Optional cleanup of NVIDIA helm repository

Examples:
  amgctl k8s remove                                # Remove deployment and delete namespace
  amgctl k8s remove --remove-repos                 # Also remove helm repositories
  amgctl k8s remove --timeout 30m                  # Custom timeout for removal
  amgctl k8s remove --timeout 1h --remove-repos    # Custom timeout with repo cleanup`,
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
	k8sRemoveCmd.PersistentFlags().Bool("delete-namespace", true, "Delete the 'amg' namespace after removing the deployment")

	// Add flags for deploy command
	k8sDeployCmd.PersistentFlags().String("timeout", "30m", "Timeout duration for helm install operation (e.g., 30m, 1h, 90m)")

	// Resource configuration flags
	k8sDeployCmd.PersistentFlags().Int("gpus", 0, "Number of GPUs to allocate per pod (default: uses chart default of 8)")
	k8sDeployCmd.PersistentFlags().Int("ib-devs", 0, "Number of InfiniBand/RDMA devices to allocate per pod (default: uses chart default of 8)")

	// RDMA configuration flags
	k8sDeployCmd.PersistentFlags().StringSlice("rdma-ifNames", []string{}, "List of RDMA interface names (e.g., ibp24s0,ibp206s0). Enables NicClusterPolicy when specified")
	k8sDeployCmd.PersistentFlags().StringSlice("rdma-device-ids", []string{}, "List of RDMA device IDs to match (empty means all)")
	k8sDeployCmd.PersistentFlags().StringSlice("rdma-drivers", []string{}, "List of RDMA drivers to match (empty means all)")
	k8sDeployCmd.PersistentFlags().StringSlice("rdma-vendors", []string{}, "List of RDMA vendor IDs to match (empty means all)")
}

// hasRDMAFlags checks if any RDMA configuration flags are provided
func hasRDMAFlags(cmd *cobra.Command) bool {
	ifNames, _ := cmd.Flags().GetStringSlice("rdma-ifNames")
	deviceIDs, _ := cmd.Flags().GetStringSlice("rdma-device-ids")
	drivers, _ := cmd.Flags().GetStringSlice("rdma-drivers")
	vendors, _ := cmd.Flags().GetStringSlice("rdma-vendors")

	return len(ifNames) > 0 || len(deviceIDs) > 0 || len(drivers) > 0 || len(vendors) > 0
}

// generateRDMASetFlags creates helm --set flags for RDMA configuration
func generateRDMASetFlags(cmd *cobra.Command) []string {
	var setFlags []string

	// Enable NicClusterPolicy
	setFlags = append(setFlags, "--set", "nicClusterPolicy.enabled=true")

	// Get RDMA configuration from flags
	ifNames, _ := cmd.Flags().GetStringSlice("rdma-ifNames")
	deviceIDs, _ := cmd.Flags().GetStringSlice("rdma-device-ids")
	drivers, _ := cmd.Flags().GetStringSlice("rdma-drivers")
	vendors, _ := cmd.Flags().GetStringSlice("rdma-vendors")

	// Set interface names if provided
	if len(ifNames) > 0 {
		ifNamesStr := "{" + strings.Join(ifNames, ",") + "}"
		setFlags = append(setFlags, "--set", "nicClusterPolicy.rdmaSharedDevicePlugin.selectors.ifNames="+ifNamesStr)
	}

	// Set device IDs if provided
	if len(deviceIDs) > 0 {
		deviceIDsStr := "{" + strings.Join(deviceIDs, ",") + "}"
		setFlags = append(setFlags, "--set", "nicClusterPolicy.rdmaSharedDevicePlugin.selectors.deviceIDs="+deviceIDsStr)
	}

	// Set drivers if provided
	if len(drivers) > 0 {
		driversStr := "{" + strings.Join(drivers, ",") + "}"
		setFlags = append(setFlags, "--set", "nicClusterPolicy.rdmaSharedDevicePlugin.selectors.drivers="+driversStr)
	}

	// Set vendors if provided
	if len(vendors) > 0 {
		vendorsStr := "{" + strings.Join(vendors, ",") + "}"
		setFlags = append(setFlags, "--set", "nicClusterPolicy.rdmaSharedDevicePlugin.selectors.vendors="+vendorsStr)
	}

	return setFlags
}

// generateResourceSetFlags creates helm --set flags for resource configuration
func generateResourceSetFlags(cmd *cobra.Command) []string {
	var setFlags []string

	// Get resource configuration from flags
	gpus, _ := cmd.Flags().GetInt("gpus")
	ibDevs, _ := cmd.Flags().GetInt("ib-devs")

	// Set GPU count if provided
	if gpus > 0 {
		setFlags = append(setFlags, "--set", fmt.Sprintf("resources.gpu=%d", gpus))
	}

	// Set InfiniBand/RDMA device count if provided
	if ibDevs > 0 {
		setFlags = append(setFlags, "--set", fmt.Sprintf("resources.rdma.count=%d", ibDevs))
	}

	return setFlags
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
func runK8sDeploy(cmd *cobra.Command) error {
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
	execCmd := exec.Command("helm", "repo", "add", "nvidia", "https://helm.ngc.nvidia.com/nvidia")
	if err := execCmd.Run(); err != nil {
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
	execCmd = exec.Command("helm", "repo", "update")
	if err := execCmd.Run(); err != nil {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("failed to update helm repositories: %w", err)
	}
	fmt.Println("✅ OK")
	fmt.Println()

	// Step 3: Install the AMG chart
	timeout, _ := cmd.Flags().GetString("timeout")
	fmt.Println("🎯 Step 3: Installing AMG chart...")

	// Check if RDMA configuration is provided
	useRDMA := hasRDMAFlags(cmd)
	chartPath := "oci://ghcr.io/sdimitro/amg-chart"

	// Check for resource configuration
	gpus, _ := cmd.Flags().GetInt("gpus")
	ibDevs, _ := cmd.Flags().GetInt("ib-devs")
	hasResourceConfig := gpus > 0 || ibDevs > 0

	if useRDMA || hasResourceConfig {
		if useRDMA {
			fmt.Println("🔧 RDMA configuration detected...")
		}
		if hasResourceConfig {
			fmt.Println("⚙️  Resource configuration detected...")
		}

		// Print configuration summary
		fmt.Println("📋 Configuration Summary:")

		// Resource configuration
		if gpus > 0 {
			fmt.Printf("   • GPUs per pod: %d\n", gpus)
		}
		if ibDevs > 0 {
			fmt.Printf("   • InfiniBand/RDMA devices per pod: %d\n", ibDevs)
		}

		// RDMA policy configuration
		if useRDMA {
			ifNames, _ := cmd.Flags().GetStringSlice("rdma-ifNames")
			deviceIDs, _ := cmd.Flags().GetStringSlice("rdma-device-ids")
			drivers, _ := cmd.Flags().GetStringSlice("rdma-drivers")
			vendors, _ := cmd.Flags().GetStringSlice("rdma-vendors")

			if len(ifNames) > 0 {
				fmt.Printf("   • RDMA Interface Names: %v\n", ifNames)
			}
			if len(deviceIDs) > 0 {
				fmt.Printf("   • RDMA Device IDs: %v\n", deviceIDs)
			}
			if len(drivers) > 0 {
				fmt.Printf("   • RDMA Drivers: %v\n", drivers)
			}
			if len(vendors) > 0 {
				fmt.Printf("   • RDMA Vendors: %v\n", vendors)
			}
			fmt.Printf("   • NicClusterPolicy: Enabled\n")
		}
	}

	fmt.Printf("This may take up to %s depending on cluster resources...\n", timeout)
	fmt.Printf("Installing AMG release (timeout: %s)... ", timeout)

	// Build helm install command
	helmArgs := []string{
		"install", "amg-release",
		chartPath,
		"--namespace", "amg",
		"--create-namespace",
		"--wait",
		"--timeout=" + timeout,
		"--debug",
	}

	// Add version for OCI chart
	helmArgs = append(helmArgs, "--version", "0.1.0")

	// Add resource configuration using --set flags if configured
	if hasResourceConfig {
		resourceSetFlags := generateResourceSetFlags(cmd)
		helmArgs = append(helmArgs, resourceSetFlags...)
	}

	// Add RDMA configuration using --set flags if configured
	if useRDMA {
		rdmaSetFlags := generateRDMASetFlags(cmd)
		helmArgs = append(helmArgs, rdmaSetFlags...)
	}

	execCmd = exec.Command("helm", helmArgs...)

	// Stream output in real-time for better user experience
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("failed to install AMG chart: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ AMG chart installed successfully!")
	fmt.Println()

	// Step 4: Verify deployment
	fmt.Println("🔍 Step 4: Verifying deployment...")
	fmt.Print("Checking AMG release status... ")
	execCmd = exec.Command("helm", "status", "amg-release", "--namespace", "amg")
	if err := execCmd.Run(); err != nil {
		fmt.Println("⚠️  WARNING - Could not verify release status")
	} else {
		fmt.Println("✅ OK")
	}

	fmt.Print("Checking AMG pods... ")
	execCmd = exec.Command("kubectl", "get", "pods", "-l", "app.kubernetes.io/instance=amg-release", "--namespace", "amg", "--no-headers")
	output, err := execCmd.Output()
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
	execCmd := exec.Command("helm", "status", "amg-release", "--namespace", "amg", "--output", "json")
	output, err := execCmd.Output()
	if err != nil {
		fmt.Println("❌ NOT FOUND")
		fmt.Println()
		fmt.Println("ℹ️  No AMG release found in 'amg' namespace. Nothing to remove.")
		fmt.Println("💡 If you're looking for a different release name, use: helm list --all-namespaces")
		return nil
	}
	fmt.Println("✅ FOUND")

	// Parse basic info about the release
	if len(output) > 0 {
		fmt.Println("📋 Release details:")
		// Get basic status without full JSON parsing
		statusCmd := exec.Command("helm", "status", "amg-release", "--namespace", "amg", "--output", "table")
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
	fmt.Println("⚠️  This will remove the AMG deployment and delete the entire 'amg' namespace by default.")
	fmt.Println()

	// Step 3: Remove the AMG helm release
	timeout, _ := cmd.Flags().GetString("timeout")
	fmt.Println("🗑️  Step 2: Removing AMG deployment...")
	fmt.Printf("Uninstalling amg-release (timeout: %s)... ", timeout)

	execCmd = exec.Command("helm", "uninstall", "amg-release", "--namespace", "amg", "--wait", "--timeout="+timeout)

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

	// Check for any remaining pods in the amg namespace
	execCmd = exec.Command("kubectl", "get", "pods", "-n", "amg", "-l", "app.kubernetes.io/instance=amg-release", "--no-headers")
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

	// Check for any remaining services in the amg namespace
	fmt.Print("Checking for remaining AMG services... ")
	execCmd = exec.Command("kubectl", "get", "services", "-n", "amg", "-l", "app.kubernetes.io/instance=amg-release", "--no-headers")
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

	// Wait for namespace to be cleaned up (check if any resources remain)
	fmt.Print("Waiting for namespace cleanup... ")
	namespaceClean := false
	for i := 0; i < 30; i++ { // Wait up to 30 seconds
		execCmd = exec.Command("kubectl", "get", "all", "-n", "amg", "--no-headers")
		output, err = execCmd.Output()
		if err != nil || strings.TrimSpace(string(output)) == "" {
			fmt.Println("✅ OK - Namespace cleaned up")
			namespaceClean = true
			break
		}
		if i == 29 {
			fmt.Println("⚠️  WARNING - Some resources may still exist in namespace")
			fmt.Println("💡 You can check with: kubectl get all -n amg")
		} else {
			time.Sleep(1 * time.Second)
		}
	}

	// Delete the namespace by default
	deleteNamespace, _ := cmd.Flags().GetBool("delete-namespace")

	if deleteNamespace {
		fmt.Print("Deleting 'amg' namespace... ")
		if !namespaceClean {
			fmt.Println("⚠️  WARNING - Namespace may still contain resources")
		}
		execCmd = exec.Command("kubectl", "delete", "namespace", "amg", "--ignore-not-found=true")
		if err := execCmd.Run(); err != nil {
			fmt.Println("❌ FAILED")
			fmt.Printf("⚠️  Warning: Failed to delete namespace: %v\n", err)
		} else {
			fmt.Println("✅ OK - Namespace deleted")
		}
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
