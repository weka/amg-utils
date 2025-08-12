package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Host environment management commands",
	Long:  `Manage the host environment setup, status, and cleanup for AMG.`,
}

var hostSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up the AMG environment",
	Long: `Set up the AMG environment by creating UV virtual environments, cloning repositories,
and installing dependencies. This replicates the functionality of setup_lmcache_stable.sh.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHostSetup()
	},
}

var hostStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show AMG environment status",
	Long:  `Display the current status of the AMG environment including UV virtual environments and repositories.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHostStatus()
	},
}

var hostClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the AMG environment",
	Long:  `Remove UV virtual environments, repositories, and other artifacts created by 'amgctl host setup'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHostClear()
	},
}

func init() {
	hostCmd.AddCommand(hostSetupCmd)
	hostCmd.AddCommand(hostStatusCmd)
	hostCmd.AddCommand(hostClearCmd)
}

// Configuration constants
const (
	uvEnvName   = "amg_stable"
	repoURL     = "git@github.com:weka/weka-LMCache.git"
	repoName    = "LMCache"
	commitHash  = "c231e2285ee61a0cbc878d51ed2e7236ac7c0b5d"
	vllmCommit  = "b6553be1bc75f046b00046a4ad7576364d03c835"
)

func getBasePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "amg_stable")
}

func getUvEnvPath() string {
	return filepath.Join(getBasePath(), ".venv")
}

func getRepoPath() string {
	return filepath.Join(getBasePath(), repoName)
}

// commandExists checks if a command is available in the system PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func runHostSetup() error {
	fmt.Println("🚀 Starting AMG environment setup...")
	
	// Handle cross-platform differences
	switch runtime.GOOS {
	case "linux":
		return runLinuxSetup()
	case "darwin":
		return runMacSetup()
	case "windows":
		return runWindowsSetup()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func runLinuxSetup() error {
	fmt.Println("🐧 Running Linux setup...")
	
	// Initial checks
	fmt.Println("--- Initial Setup Checks ---")
	if !commandExists("uv") {
		return fmt.Errorf("uv command not found. Please install uv: https://docs.astral.sh/uv/getting-started/installation/")
	}
	
	if !commandExists("git") {
		return fmt.Errorf("git command not found. Please install Git")
	}
	
	fmt.Println("✅ uv and Git commands found. Proceeding with setup.")
	
	// Check and create uv virtual environment
	if err := setupUvEnvironment(); err != nil {
		return fmt.Errorf("failed to setup uv environment: %w", err)
	}
	
	// Setup repository
	if err := setupRepository(); err != nil {
		return fmt.Errorf("failed to setup repository: %w", err)
	}
	
	fmt.Println("🎉 Setup completed successfully!")
	return nil
}

func runMacSetup() error {
	fmt.Println("🍎 Mac setup not yet implemented. This is a placeholder.")
	fmt.Println("The Mac implementation will include:")
	fmt.Println("  - Homebrew dependency checks")
	fmt.Println("  - macOS-specific UV setup")
	fmt.Println("  - Platform-specific optimizations")
	return nil
}

func runWindowsSetup() error {
	fmt.Println("🪟 Windows setup not yet implemented. This is a placeholder.")
	fmt.Println("The Windows implementation will include:")
	fmt.Println("  - PowerShell/cmd compatibility")
	fmt.Println("  - Windows-specific path handling")
	fmt.Println("  - UV package manager integration")
	return nil
}

func setupUvEnvironment() error {
	fmt.Println("\n--- UV Virtual Environment Setup ---")
	
	basePath := getBasePath()
	uvEnvPath := getUvEnvPath()
	
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create base path '%s': %w", basePath, err)
	}
	
	fmt.Printf("Checking for UV virtual environment: '%s'...\n", uvEnvPath)
	
	// Check if uv virtual environment exists
	if _, err := os.Stat(uvEnvPath); os.IsNotExist(err) {
		fmt.Printf("UV virtual environment '%s' not found.\n", uvEnvPath)
		fmt.Printf("Creating UV virtual environment '%s' with Python 3.12...\n", uvEnvName)
		
		// Navigate to the base path and create the virtual environment
		cmd := exec.Command("uv", "venv", "--python", "3.12", ".venv")
		cmd.Dir = basePath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create uv virtual environment '%s': %w", uvEnvName, err)
		}
		
		fmt.Printf("✅ UV virtual environment '%s' created successfully.\n", uvEnvName)
		
		// Install packages for new environment
		if err := installUvPackages(); err != nil {
			return fmt.Errorf("failed to install uv packages: %w", err)
		}
	} else {
		fmt.Printf("✅ UV virtual environment '%s' already exists.\n", uvEnvName)
	}
	
	return nil
}

func installUvPackages() error {
	fmt.Println("Installing initial Python packages...")
	
	basePath := getBasePath()
	packages := []string{
		fmt.Sprintf("https://wheels.vllm.ai/%s/vllm-1.0.0.dev-cp38-abi3-manylinux1_x86_64.whl", vllmCommit),
		"py-spy",
		"scalene",
		"pyinstrument",
		"line_profiler",
	}
	
	for _, pkg := range packages {
		fmt.Printf("Installing %s...\n", pkg)
		cmd := exec.Command("uv", "pip", "install", "--no-cache-dir", pkg)
		cmd.Dir = basePath
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️ Warning: Failed to install %s\n", pkg)
		} else {
			fmt.Printf("✅ Installed %s successfully\n", pkg)
		}
	}
	
	return nil
}

func setupRepository() error {
	fmt.Println("\n--- GitHub Repository Setup ---")
	
	basePath := getBasePath()
	repoPath := getRepoPath()
	
	// Create base directory
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create base path '%s': %w", basePath, err)
	}
	
	// Check if repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		fmt.Printf("Repository directory '%s' not found.\n", repoPath)
		fmt.Printf("Cloning repository '%s' into '%s'...\n", repoURL, repoPath)
		
		cmd := exec.Command("git", "clone", repoURL, repoPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		
		fmt.Println("✅ Repository cloned successfully.")
	} else {
		fmt.Printf("Repository directory '%s' found.\n", repoPath)
		fmt.Println("Pulling latest changes...")
		
		cmd := exec.Command("git", "-C", repoPath, "pull")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to pull repository changes: %w", err)
		}
		
		fmt.Println("✅ Repository updated.")
	}
	
	// Checkout specific commit
	if err := checkoutCommit(repoPath); err != nil {
		return fmt.Errorf("failed to checkout commit: %w", err)
	}
	
	// Install repository dependencies
	if err := installRepositoryDependencies(repoPath); err != nil {
		return fmt.Errorf("failed to install repository dependencies: %w", err)
	}
	
	return nil
}

func checkoutCommit(repoPath string) error {
	fmt.Println("\n--- Git Commit Checkout ---")
	
	// Get current commit
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}
	
	currentCommit := strings.TrimSpace(string(output))
	
	if currentCommit != commitHash {
		fmt.Printf("Current commit (%s) does not match target commit (%s).\n", currentCommit, commitHash)
		fmt.Printf("Checking out commit: %s...\n", commitHash)
		
		cmd := exec.Command("git", "-C", repoPath, "checkout", commitHash)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout commit '%s': %w", commitHash, err)
		}
		
		fmt.Printf("✅ Successfully checked out commit: %s\n", commitHash)
	} else {
		fmt.Printf("✅ Repository is already at the target commit: %s\n", commitHash)
	}
	
	return nil
}

func installRepositoryDependencies(repoPath string) error {
	fmt.Println("\n--- Installing Repository Dependencies ---")
	
	reqFiles := []string{
		filepath.Join(repoPath, "requirements", "build.txt"),
		filepath.Join(repoPath, "requirements", "common.txt"),
		filepath.Join(repoPath, "requirements", "cuda.txt"),
	}
	
	// Check if requirement files exist
	allExist := true
	for _, reqFile := range reqFiles {
		if _, err := os.Stat(reqFile); os.IsNotExist(err) {
			allExist = false
			break
		}
	}
	
	if allExist {
		fmt.Println("Installing dependencies from requirements files...")
		args := []string{"pip", "install", "--no-cache-dir"}
		for _, reqFile := range reqFiles {
			args = append(args, "-r", reqFile)
		}
		
		cmd := exec.Command("uv", args...)
		cmd.Dir = repoPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Println("⚠️ Warning: Failed to install repository dependencies")
		} else {
			fmt.Println("✅ Repository dependencies installed successfully")
		}
	} else {
		fmt.Println("⚠️ One or more requirement files not found. Skipping dependency installation.")
	}
	
	// Install in editable mode
	fmt.Println("Installing repository in editable mode...")
	cmd := exec.Command("uv", "pip", "install", "-e", ".")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		fmt.Println("⚠️ Warning: Failed to install repository in editable mode")
	} else {
		fmt.Println("✅ Repository installed in editable mode successfully")
	}
	
	// Hot-patch transformers
	fmt.Println("Hot-patching transformers package...")
	cmd = exec.Command("uv", "pip", "install", "--no-cache-dir", "transformers<4.54.0")
	cmd.Dir = repoPath
	
	if err := cmd.Run(); err != nil {
		fmt.Println("⚠️ Warning: Failed to hot-patch transformers package")
	} else {
		fmt.Println("✅ Downgraded transformers explicitly")
	}
	
	return nil
}

func runHostStatus() error {
	fmt.Println("📊 AMG Environment Status")
	fmt.Println("This is a placeholder for host status functionality.")
	fmt.Println("Will show:")
	fmt.Println("  - UV virtual environment status")
	fmt.Println("  - Repository status and commit")
	fmt.Println("  - Installed packages")
	fmt.Println("  - System resources")
	return nil
}

func runHostClear() error {
	fmt.Println("🧹 Clearing AMG environment...")
	
	// Handle cross-platform differences
	switch runtime.GOOS {
	case "linux":
		return runLinuxClear()
	case "darwin":
		return runMacClear()
	case "windows":
		return runWindowsClear()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func runLinuxClear() error {
	fmt.Println("🐧 Running Linux cleanup...")
	
	// Remove UV virtual environment (by removing the base directory which contains .venv)
	basePath := getBasePath()
	if _, err := os.Stat(basePath); err == nil {
		fmt.Printf("Removing UV environment and directory '%s'...\n", basePath)
		if err := os.RemoveAll(basePath); err != nil {
			fmt.Printf("⚠️ Warning: Failed to remove directory '%s': %v\n", basePath, err)
		} else {
			fmt.Printf("✅ Directory '%s' (including UV virtual environment) removed successfully\n", basePath)
		}
	} else {
		fmt.Printf("Directory '%s' does not exist\n", basePath)
	}
	
	fmt.Println("🎉 Cleanup completed!")
	return nil
}

func runMacClear() error {
	fmt.Println("🍎 Mac cleanup not yet implemented. This is a placeholder.")
	fmt.Println("The Mac implementation will include:")
	fmt.Println("  - Homebrew cleanup")
	fmt.Println("  - macOS-specific file removal")
	fmt.Println("  - UV virtual environment cleanup")
	return nil
}

func runWindowsClear() error {
	fmt.Println("🪟 Windows cleanup not yet implemented. This is a placeholder.")
	fmt.Println("The Windows implementation will include:")
	fmt.Println("  - Windows-specific cleanup")
	fmt.Println("  - Registry cleanup if needed")
	fmt.Println("  - UV virtual environment cleanup")
	return nil
}
