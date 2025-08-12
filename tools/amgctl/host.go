package main

import (
	"encoding/json"
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
		return runHostSetup(cmd)
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

var hostUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update LMCache to latest commit when following a branch",
	Long:  `Update LMCache repository to the latest commit of the current branch. Only works when LMCache was installed following a branch instead of a specific commit.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHostUpdate()
	},
}

func init() {
	hostCmd.AddCommand(hostSetupCmd)
	hostCmd.AddCommand(hostStatusCmd)
	hostCmd.AddCommand(hostClearCmd)
	hostCmd.AddCommand(hostUpdateCmd)

	// Add flags to hostSetupCmd
	hostSetupCmd.Flags().Bool("skip-hotfixes", false, "Skip applying hotfixes like downgrading transformers")
	hostSetupCmd.Flags().String("lmcache-repo", repoURL, "Alternative LMCache repository URL")
	hostSetupCmd.Flags().String("lmcache-commit", commitHash, "Specific commit hash for LMCache repository")
	hostSetupCmd.Flags().String("lmcache-branch", "", "Branch to follow for LMCache repository (overrides commit)")
	hostSetupCmd.Flags().String("vllm-commit", vllmCommit, "Alternative vLLM commit hash")
}

// Configuration constants
const (
	uvEnvName  = "amg_stable"
	repoURL    = "git@github.com:weka/weka-LMCache.git"
	repoName   = "LMCache"
	commitHash = "c231e2285ee61a0cbc878d51ed2e7236ac7c0b5d"
	vllmCommit = "b6553be1bc75f046b00046a4ad7576364d03c835"
	stateFile  = "amg_setup_state.json"
)

// SetupState tracks the configuration used during setup
type SetupState struct {
	LMCacheRepo   string `json:"lmcache_repo"`
	LMCacheCommit string `json:"lmcache_commit,omitempty"`
	LMCacheBranch string `json:"lmcache_branch,omitempty"`
	VLLMCommit    string `json:"vllm_commit"`
	SkipHotfixes  bool   `json:"skip_hotfixes"`
}

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

func getStateFilePath() string {
	return filepath.Join(getBasePath(), stateFile)
}

// saveSetupState saves the setup configuration to a JSON file
func saveSetupState(state *SetupState) error {
	stateFilePath := getStateFilePath()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal setup state: %w", err)
	}

	if err := os.WriteFile(stateFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write setup state file: %w", err)
	}

	return nil
}

// loadSetupState loads the setup configuration from the JSON file
func loadSetupState() (*SetupState, error) {
	stateFilePath := getStateFilePath()
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No state file exists
		}
		return nil, fmt.Errorf("failed to read setup state file: %w", err)
	}

	var state SetupState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal setup state: %w", err)
	}

	return &state, nil
}

// commandExists checks if a command is available in the system PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func runHostSetup(cmd *cobra.Command) error {
	fmt.Println("🚀 Starting AMG environment setup...")

	// Get flag values
	skipHotfixes, _ := cmd.Flags().GetBool("skip-hotfixes")
	lmcacheRepo, _ := cmd.Flags().GetString("lmcache-repo")
	lmcacheCommit, _ := cmd.Flags().GetString("lmcache-commit")
	lmcacheBranch, _ := cmd.Flags().GetString("lmcache-branch")
	vllmCommitFlag, _ := cmd.Flags().GetString("vllm-commit")

	// Create setup state
	state := &SetupState{
		LMCacheRepo:   lmcacheRepo,
		LMCacheCommit: lmcacheCommit,
		LMCacheBranch: lmcacheBranch,
		VLLMCommit:    vllmCommitFlag,
		SkipHotfixes:  skipHotfixes,
	}

	// If branch is specified, clear commit to indicate we're following a branch
	if lmcacheBranch != "" {
		state.LMCacheCommit = ""
	}

	// Handle cross-platform differences
	switch runtime.GOOS {
	case "linux":
		return runLinuxSetup(state)
	case "darwin":
		return runMacSetup(state)
	case "windows":
		return runWindowsSetup(state)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func runLinuxSetup(state *SetupState) error {
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
	if err := setupUvEnvironment(state); err != nil {
		return fmt.Errorf("failed to setup uv environment: %w", err)
	}

	// Setup repository
	if err := setupRepository(state); err != nil {
		return fmt.Errorf("failed to setup repository: %w", err)
	}

	// Save setup state
	if err := saveSetupState(state); err != nil {
		fmt.Printf("⚠️ Warning: Failed to save setup state: %v\n", err)
	}

	fmt.Println("🎉 Setup completed successfully!")
	return nil
}

func runMacSetup(state *SetupState) error {
	fmt.Println("🍎 Mac setup not yet implemented. This is a placeholder.")
	fmt.Println("The Mac implementation will include:")
	fmt.Println("  - Homebrew dependency checks")
	fmt.Println("  - macOS-specific UV setup")
	fmt.Println("  - Platform-specific optimizations")
	return nil
}

func runWindowsSetup(state *SetupState) error {
	fmt.Println("🪟 Windows setup not yet implemented. This is a placeholder.")
	fmt.Println("The Windows implementation will include:")
	fmt.Println("  - PowerShell/cmd compatibility")
	fmt.Println("  - Windows-specific path handling")
	fmt.Println("  - UV package manager integration")
	return nil
}

func setupUvEnvironment(state *SetupState) error {
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
		if err := installUvPackages(state); err != nil {
			return fmt.Errorf("failed to install uv packages: %w", err)
		}
	} else {
		fmt.Printf("✅ UV virtual environment '%s' already exists.\n", uvEnvName)
	}

	return nil
}

func installUvPackages(state *SetupState) error {
	fmt.Println("Installing initial Python packages...")

	basePath := getBasePath()
	packages := []string{
		fmt.Sprintf("https://wheels.vllm.ai/%s/vllm-1.0.0.dev-cp38-abi3-manylinux1_x86_64.whl", state.VLLMCommit),
		"py-spy",
		"scalene",
		"pyinstrument",
		"line_profiler",
		"fastsafetensors",
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

func setupRepository(state *SetupState) error {
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
		fmt.Printf("Cloning repository '%s' into '%s'...\n", state.LMCacheRepo, repoPath)

		cmd := exec.Command("git", "clone", state.LMCacheRepo, repoPath)
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

	// Checkout specific commit or branch
	if err := checkoutCommitOrBranch(repoPath, state); err != nil {
		return fmt.Errorf("failed to checkout commit/branch: %w", err)
	}

	// Install repository dependencies
	if err := installRepositoryDependencies(repoPath, state); err != nil {
		return fmt.Errorf("failed to install repository dependencies: %w", err)
	}

	return nil
}

func checkoutCommitOrBranch(repoPath string, state *SetupState) error {
	fmt.Println("\n--- Git Checkout ---")

	if state.LMCacheBranch != "" {
		// Branch mode - checkout and track the branch
		fmt.Printf("Checking out branch: %s...\n", state.LMCacheBranch)

		// First, fetch all branches
		cmd := exec.Command("git", "-C", repoPath, "fetch", "origin")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to fetch origin: %w", err)
		}

		// Checkout the branch
		cmd = exec.Command("git", "-C", repoPath, "checkout", "-B", state.LMCacheBranch, "origin/"+state.LMCacheBranch)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout branch '%s': %w", state.LMCacheBranch, err)
		}

		fmt.Printf("✅ Successfully checked out and tracking branch: %s\n", state.LMCacheBranch)
	} else if state.LMCacheCommit != "" {
		// Commit mode - checkout specific commit
		fmt.Printf("Checking out commit: %s...\n", state.LMCacheCommit)

		// Get current commit
		cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get current commit: %w", err)
		}

		currentCommit := strings.TrimSpace(string(output))

		if currentCommit != state.LMCacheCommit {
			fmt.Printf("Current commit (%s) does not match target commit (%s).\n", currentCommit, state.LMCacheCommit)

			cmd := exec.Command("git", "-C", repoPath, "checkout", state.LMCacheCommit)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to checkout commit '%s': %w", state.LMCacheCommit, err)
			}

			fmt.Printf("✅ Successfully checked out commit: %s\n", state.LMCacheCommit)
		} else {
			fmt.Printf("✅ Repository is already at the target commit: %s\n", state.LMCacheCommit)
		}
	}

	return nil
}

func installRepositoryDependencies(repoPath string, state *SetupState) error {
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

	// Apply hotfixes unless skipped
	if !state.SkipHotfixes {
		fmt.Println("Hot-patching transformers package...")
		cmd = exec.Command("uv", "pip", "install", "--no-cache-dir", "transformers<4.54.0")
		cmd.Dir = repoPath

		if err := cmd.Run(); err != nil {
			fmt.Println("⚠️ Warning: Failed to hot-patch transformers package")
		} else {
			fmt.Println("✅ Downgraded transformers explicitly")
		}
	} else {
		fmt.Println("🚫 Skipping hotfixes (transformers downgrade) as requested")
	}

	return nil
}

func runHostUpdate() error {
	fmt.Println("🔄 Updating LMCache repository...")

	// Load the setup state to check if we're following a branch
	state, err := loadSetupState()
	if err != nil {
		return fmt.Errorf("failed to load setup state: %w", err)
	}

	if state == nil {
		return fmt.Errorf("no setup state found. Please run 'amgctl host setup' first")
	}

	if state.LMCacheBranch == "" {
		return fmt.Errorf("LMCache is not configured to follow a branch. Update is only available when following a branch instead of a specific commit")
	}

	repoPath := getRepoPath()

	// Check if repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return fmt.Errorf("LMCache repository not found at '%s'. Please run 'amgctl host setup' first", repoPath)
	}

	fmt.Printf("Updating LMCache repository to latest commit of branch '%s'...\n", state.LMCacheBranch)

	// Fetch latest changes
	cmd := exec.Command("git", "-C", repoPath, "fetch", "origin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch latest changes: %w", err)
	}

	// Get current commit before update
	cmd = exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	beforeOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}
	beforeCommit := strings.TrimSpace(string(beforeOutput))

	// Pull latest changes for the branch
	cmd = exec.Command("git", "-C", repoPath, "pull", "origin", state.LMCacheBranch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull latest changes: %w", err)
	}

	// Get current commit after update
	cmd = exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	afterOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get updated commit: %w", err)
	}
	afterCommit := strings.TrimSpace(string(afterOutput))

	if beforeCommit == afterCommit {
		fmt.Printf("✅ Repository is already up to date at commit: %s\n", afterCommit)
	} else {
		fmt.Printf("✅ Repository updated from %s to %s\n", beforeCommit[:8], afterCommit[:8])

		// Reinstall repository dependencies to pick up any changes
		fmt.Println("Reinstalling repository dependencies...")
		if err := installRepositoryDependencies(repoPath, state); err != nil {
			fmt.Printf("⚠️ Warning: Failed to reinstall repository dependencies: %v\n", err)
		}
	}

	fmt.Println("🎉 Update completed successfully!")
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
