package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Host environment management commands",
	Long:  `Manage and monitor the host environment for AMG.`,
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
	Long:  `Check the status of the host environment, including required software.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		return runHostStatus(verbose)
	},
}

var hostClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the AMG environment",
	Long:  `Remove UV virtual environments, repositories, and other artifacts created by 'amgctl host setup'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHostClear(cmd)
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

var hostPreFlightCmd = &cobra.Command{
	Use:   "pre-flight",
	Short: "Verify system readiness for AMG setup and execution",
	Long:  `Perform pre-flight checks to ensure the host environment is ready for AMG setup and execution. This includes validating required tools, configurations, and system settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		full, _ := cmd.Flags().GetBool("full")
		return runHostPreFlight(full)
	},
}

func init() {
	hostCmd.AddCommand(hostSetupCmd)
	hostCmd.AddCommand(hostStatusCmd)
	hostCmd.AddCommand(hostClearCmd)
	hostCmd.AddCommand(hostUpdateCmd)
	hostCmd.AddCommand(hostPreFlightCmd)

	// Add flags to hostSetupCmd
	hostSetupCmd.Flags().String("lmcache-repo", repoURL, "Alternative LMCache repository URL")
	hostSetupCmd.Flags().String("lmcache-commit", "", "Specific commit hash for LMCache repository")
	hostSetupCmd.Flags().String("lmcache-branch", defaultRef, "Branch or tag to follow for LMCache repository (overrides commit)")
	hostSetupCmd.Flags().String("vllm-version", vllmVersion, "vLLM version to install (e.g., 0.9.2, 0.10.0)")

	// Add flags to hostStatusCmd
	hostStatusCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	// Add flags to hostPreFlightCmd
	hostPreFlightCmd.Flags().Bool("full", false, "Run comprehensive checks including GPU Direct Storage (GDS) validation")

	// Add flags to hostClearCmd
	hostClearCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt and proceed with deletion")
}

// Configuration constants
const (
	uvEnvName   = "amg_stable"
	repoURL     = "https://github.com/LMCache/LMCache.git"
	repoName    = "LMCache"
	defaultRef  = "v0.3.3" // Can be a tag or branch
	vllmVersion = "0.10.0"
	stateFile   = ".amg_setup_state.json"
)

// SetupState tracks the configuration used during setup
type SetupState struct {
	LMCacheRepo   string `json:"lmcache_repo"`
	LMCacheCommit string `json:"lmcache_commit,omitempty"`
	LMCacheBranch string `json:"lmcache_branch,omitempty"`
	VLLMVersion   string `json:"vllm_version"`
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

// CuFileConfig represents the structure of /etc/cufile.json
type CuFileConfig struct {
	Execution struct {
		MaxIOThreads int `json:"max_io_threads"`
	} `json:"execution"`
}

// runHostSystemChecks performs shared system checks for both setup and pre-flight commands
func runHostSystemChecks() error {
	fmt.Println("--- System Checks ---")

	// Check for required commands
	if !commandExists("uv") {
		return fmt.Errorf("uv command not found. Please install uv: https://docs.astral.sh/uv/getting-started/installation/")
	}
	fmt.Println("✅ uv command found")

	if !commandExists("git") {
		return fmt.Errorf("git command not found. Please install Git")
	}
	fmt.Println("✅ git command found")

	// Check cufile.json configuration
	if err := checkCuFileConfig(); err != nil {
		// This is a warning, not a fatal error
		fmt.Printf("⚠️  %v\n", err)
	}

	fmt.Println("✅ System checks completed")
	return nil
}

// stripJSONComments removes C-style comments from JSON content
func stripJSONComments(jsonData []byte) []byte {
	// Remove single-line comments (//)
	singleLineCommentRe := regexp.MustCompile(`//.*`)
	result := singleLineCommentRe.ReplaceAll(jsonData, []byte(""))

	// Remove multi-line comments (/* */)
	multiLineCommentRe := regexp.MustCompile(`(?s)/\*.*?\*/`)
	result = multiLineCommentRe.ReplaceAll(result, []byte(""))

	return result
}

// checkCuFileConfig validates /etc/cufile.json configuration
func checkCuFileConfig() error {
	cufilePath := "/etc/cufile.json"

	// Check if file exists
	if _, err := os.Stat(cufilePath); os.IsNotExist(err) {
		return fmt.Errorf("cufile.json not found at %s. Consider configuring CUDA file operations if needed", cufilePath)
	}

	// Read the file
	data, err := os.ReadFile(cufilePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", cufilePath, err)
	}

	// Strip comments from JSON content
	cleanData := stripJSONComments(data)

	// Parse JSON
	var config CuFileConfig
	if err := json.Unmarshal(cleanData, &config); err != nil {
		return fmt.Errorf("failed to parse %s: %w", cufilePath, err)
	}

	// Check max_io_threads
	if config.Execution.MaxIOThreads != 0 {
		return fmt.Errorf("cufile.json warning: execution.max_io_threads is set to %d, but should be 0 for optimal performance", config.Execution.MaxIOThreads)
	}

	fmt.Println("✅ cufile.json configuration is optimal (execution.max_io_threads = 0)")
	return nil
}

// isCondaActive checks if a conda environment is currently active
func isCondaActive() bool {
	condaEnv := os.Getenv("CONDA_DEFAULT_ENV")
	condaPrefix := os.Getenv("CONDA_PREFIX")
	return condaEnv != "" || condaPrefix != ""
}

// checkCondaDeactivated ensures no conda environment is active
func checkCondaDeactivated() error {
	if isCondaActive() {
		return fmt.Errorf("conda environment is currently active. Please deactivate your conda environment before using amgctl host commands:\n  conda deactivate")
	}
	return nil
}

// askForConfirmation prompts the user for a yes/no confirmation
func askForConfirmation(prompt string) (bool, error) {
	fmt.Printf("%s (y/N): ", prompt)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// customizeActivationScript modifies the virtual environment activation script
// to show "(amg)" instead of "(.venv)" in the prompt
func customizeActivationScript(uvEnvPath string) error {
	activateScript := filepath.Join(uvEnvPath, "bin", "activate")

	// Read the current activation script
	content, err := os.ReadFile(activateScript)
	if err != nil {
		return fmt.Errorf("failed to read activation script: %w", err)
	}

	contentStr := string(content)

	// The UV activation script has a conditional that determines VIRTUAL_ENV_PROMPT
	// We need to fix the condition to always use our custom prompt
	// Look for the pattern: if [ "x" != x ] ; then
	// and replace it with: if [ "x" = "x" ] ; then
	// This ensures the custom prompt is always used
	if strings.Contains(contentStr, `if [ "x" != x ] ; then`) {
		contentStr = strings.ReplaceAll(contentStr, `if [ "x" != x ] ; then`, `if [ "x" = "x" ] ; then`)
	}

	// Also ensure the VIRTUAL_ENV_PROMPT is set to "amg" in the true branch
	// Look for VIRTUAL_ENV_PROMPT="..." pattern and replace with our value
	lines := strings.Split(contentStr, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, `VIRTUAL_ENV_PROMPT="`) && !strings.Contains(line, "#") {
			lines[i] = strings.Replace(line, trimmed, `VIRTUAL_ENV_PROMPT="amg"`, 1)
			break
		}
	}
	contentStr = strings.Join(lines, "\n")

	// Write the modified content back
	err = os.WriteFile(activateScript, []byte(contentStr), 0755)
	if err != nil {
		return fmt.Errorf("failed to write modified activation script: %w", err)
	}

	return nil
}

func runHostSetup(cmd *cobra.Command) error {
	// Check that conda is not active
	if err := checkCondaDeactivated(); err != nil {
		return err
	}

	fmt.Println("🚀 Starting AMG environment setup...")

	// Get flag values
	lmcacheRepo, _ := cmd.Flags().GetString("lmcache-repo")
	lmcacheCommit, _ := cmd.Flags().GetString("lmcache-commit")
	lmcacheBranch, _ := cmd.Flags().GetString("lmcache-branch")
	vllmVersionFlag, _ := cmd.Flags().GetString("vllm-version")

	// Create setup state
	state := &SetupState{
		LMCacheRepo:   lmcacheRepo,
		LMCacheCommit: lmcacheCommit,
		LMCacheBranch: lmcacheBranch,
		VLLMVersion:   vllmVersionFlag,
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

	// Run shared system checks
	if err := runHostSystemChecks(); err != nil {
		return err
	}

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
	fmt.Println()
	fmt.Println("📋 Next Steps:")
	fmt.Println("  1. Navigate to the AMG environment directory:")
	fmt.Printf("     cd %s\n", getBasePath())
	fmt.Println("  2. Activate the virtual environment:")
	fmt.Println("     source .venv/bin/activate")
	fmt.Println("  3. Your shell prompt will show '(amg)' when the environment is active")
	fmt.Println("  4. To deactivate later, simply run: deactivate")
	fmt.Println()
	fmt.Println("🚀 You're ready to use the AMG environment!")
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

		// Customize the activation script to show "(amg)" instead of "(.venv)"
		if err := customizeActivationScript(uvEnvPath); err != nil {
			fmt.Printf("⚠️ Warning: Failed to customize activation script: %v\n", err)
		}

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

	// Install vLLM with specified version (torch will be automatically installed as dependency)
	vllmPackage := fmt.Sprintf("vllm==%s", state.VLLMVersion)
	fmt.Printf("Installing vLLM version %s (including torch dependencies)...\n", state.VLLMVersion)
	// Add --torch-backend=auto to auto-detect the optimial pytorch backend for us
	cmd := exec.Command("uv", "pip", "install", "--no-cache-dir", vllmPackage, "--torch-backend=auto")
	cmd.Dir = basePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install vLLM version %s: %w", state.VLLMVersion, err)
	}
	fmt.Printf("✅ vLLM version %s installed successfully\n", state.VLLMVersion)

	// Install other packages
	otherPackages := []string{
		"py-spy",
		"scalene",
		"pyinstrument",
		"line_profiler",
		"fastsafetensors",
	}

	for _, pkg := range otherPackages {
		fmt.Printf("Installing %s...\n", pkg)
		cmd := exec.Command("uv", "pip", "install", "--no-cache-dir", pkg)
		cmd.Dir = basePath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️ Warning: Failed to install %s: %v\n", pkg, err)
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

// isTag checks if the given reference is a tag
func isTag(repoPath, ref string) bool {
	cmd := exec.Command("git", "-C", repoPath, "tag", "-l", ref)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == ref
}

// isBranch checks if the given reference is a remote branch
func isBranch(repoPath, ref string) bool {
	cmd := exec.Command("git", "-C", repoPath, "ls-remote", "--heads", "origin", ref)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func checkoutCommitOrBranch(repoPath string, state *SetupState) error {
	fmt.Println("\n--- Git Checkout ---")

	if state.LMCacheBranch != "" {
		// First, fetch all references (branches and tags)
		cmd := exec.Command("git", "-C", repoPath, "fetch", "origin", "--tags")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to fetch origin: %w", err)
		}

		// Check if it's a tag or branch
		if isTag(repoPath, state.LMCacheBranch) {
			// Tag mode - checkout the tag directly
			fmt.Printf("Checking out tag: %s...\n", state.LMCacheBranch)

			cmd = exec.Command("git", "-C", repoPath, "checkout", state.LMCacheBranch)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to checkout tag '%s': %w", state.LMCacheBranch, err)
			}

			fmt.Printf("✅ Successfully checked out tag: %s\n", state.LMCacheBranch)
		} else if isBranch(repoPath, state.LMCacheBranch) {
			// Branch mode - checkout and track the branch
			fmt.Printf("Checking out branch: %s...\n", state.LMCacheBranch)

			cmd = exec.Command("git", "-C", repoPath, "checkout", "-B", state.LMCacheBranch, "origin/"+state.LMCacheBranch)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to checkout branch '%s': %w", state.LMCacheBranch, err)
			}

			fmt.Printf("✅ Successfully checked out and tracking branch: %s\n", state.LMCacheBranch)
		} else {
			return fmt.Errorf("reference '%s' is neither a valid tag nor a remote branch", state.LMCacheBranch)
		}
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
		args := []string{"pip", "install", "--no-cache-dir", "--no-build-isolation"}
		for _, reqFile := range reqFiles {
			args = append(args, "-r", reqFile)
		}

		cmd := exec.Command("uv", args...)
		cmd.Dir = repoPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️ Warning: Failed to install repository dependencies: %v\n", err)
		} else {
			fmt.Println("✅ Repository dependencies installed successfully")
		}
	} else {
		fmt.Println("⚠️ One or more requirement files not found. Skipping dependency installation.")
	}

	// Install in editable mode with --no-build-isolation to avoid xformers build issues
	fmt.Println("Installing repository in editable mode...")
	cmd := exec.Command("uv", "pip", "install", "-e", ".", "--no-build-isolation")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to install repository in editable mode: %v\n", err)
	} else {
		fmt.Println("✅ Repository installed in editable mode successfully")
	}

	return nil
}

func runHostUpdate() error {
	// Check that conda is not active
	if err := checkCondaDeactivated(); err != nil {
		return err
	}

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

func runHostPreFlight(full bool) error {
	if full {
		fmt.Println("🔍 Running comprehensive AMG pre-flight checks...")
	} else {
		fmt.Println("🔍 Running AMG pre-flight checks...")
	}
	fmt.Println()

	// Check that conda is not active
	if err := checkCondaDeactivated(); err != nil {
		return err
	}

	// Run system checks
	if err := runHostSystemChecks(); err != nil {
		return err
	}

	// Run GDS checks if --full flag is enabled
	if full {
		fmt.Println()
		if err := runGDSChecks(); err != nil {
			return fmt.Errorf("GDS checks failed: %w", err)
		}
	}

	fmt.Println()
	fmt.Println("🎉 Pre-flight checks completed successfully!")
	fmt.Println()
	fmt.Println("📋 Next Steps:")
	fmt.Println("  • Your system is ready for AMG setup")
	fmt.Println("  • Run 'amgctl host setup' to install and configure AMG")
	fmt.Println("  • Run 'amgctl host status' to check environment status")

	return nil
}

// runGDSChecks performs GPU Direct Storage checks using gdscheck
func runGDSChecks() error {
	fmt.Println("--- GPU Direct Storage (GDS) Checks ---")

	gdsCheckPath := "/usr/local/cuda/gds/tools/gdscheck"

	// Check if gdscheck tool exists
	if _, err := os.Stat(gdsCheckPath); os.IsNotExist(err) {
		return fmt.Errorf("gdscheck tool not found at %s. GPU Direct Storage may not be installed", gdsCheckPath)
	}

	fmt.Printf("✅ Found gdscheck tool at %s\n", gdsCheckPath)
	fmt.Println("Running GDS platform checks...")

	// Run gdscheck -p
	cmd := exec.Command(gdsCheckPath, "-p")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run gdscheck: %w", err)
	}

	outputStr := string(output)

	// Parse and validate the output
	if err := validateGDSOutput(outputStr); err != nil {
		return err
	}

	// Check for gdsio tool (warning only)
	gdsioPath := "/usr/local/cuda/gds/tools/gdsio"
	if _, err := os.Stat(gdsioPath); os.IsNotExist(err) {
		fmt.Printf("⚠️  gdsio tool not found at %s. Consider installing GDS IO utilities for performance testing\n", gdsioPath)
	} else {
		fmt.Printf("✅ Found gdsio tool at %s\n", gdsioPath)
	}

	fmt.Println("✅ GDS checks completed successfully")
	return nil
}

// validateGDSOutput parses gdscheck output and validates required components
func validateGDSOutput(output string) error {
	lines := strings.Split(output, "\n")

	// Track requirements
	requirements := map[string]bool{
		"nvme_supported":              false,
		"wekafs_supported":            false,
		"userspace_rdma_supported":    false,
		"mellanox_peerdirect_enabled": false,
		"rdma_library_loaded":         false,
		"rdma_devices_configured":     false,
		"iommu_disabled":              false,
	}

	// Parse the output line by line
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check NVMe support
		if strings.Contains(line, "NVMe") && strings.Contains(line, ": Supported") {
			requirements["nvme_supported"] = true
		}

		// Check WekaFS support
		if strings.Contains(line, "WekaFS") && strings.Contains(line, ": Supported") {
			requirements["wekafs_supported"] = true
		}

		// Check Userspace RDMA support
		if strings.Contains(line, "Userspace RDMA") && strings.Contains(line, ": Supported") {
			requirements["userspace_rdma_supported"] = true
		}

		// Check Mellanox PeerDirect
		if strings.Contains(line, "--Mellanox PeerDirect") && strings.Contains(line, ": Enabled") {
			requirements["mellanox_peerdirect_enabled"] = true
		}

		// Check rdma library
		if strings.Contains(line, "--rdma library") && strings.Contains(line, ": Loaded") {
			requirements["rdma_library_loaded"] = true
		}

		// Check rdma devices
		if strings.Contains(line, "--rdma devices") && strings.Contains(line, ": Configured") {
			requirements["rdma_devices_configured"] = true
		}

		// Check IOMMU status
		if strings.Contains(line, "IOMMU: disabled") {
			requirements["iommu_disabled"] = true
		}
	}

	// Validate all requirements
	var errors []string

	if !requirements["nvme_supported"] {
		errors = append(errors, "NVMe is not supported")
	} else {
		fmt.Println("✅ NVMe: Supported")
	}

	if !requirements["wekafs_supported"] {
		errors = append(errors, "WekaFS is not supported")
	} else {
		fmt.Println("✅ WekaFS: Supported")
	}

	if !requirements["userspace_rdma_supported"] {
		errors = append(errors, "Userspace RDMA is not supported")
	} else {
		fmt.Println("✅ Userspace RDMA: Supported")
	}

	if !requirements["mellanox_peerdirect_enabled"] {
		errors = append(errors, "Mellanox PeerDirect is not enabled")
	} else {
		fmt.Println("✅ Mellanox PeerDirect: Enabled")
	}

	if !requirements["rdma_library_loaded"] {
		errors = append(errors, "RDMA library is not loaded")
	} else {
		fmt.Println("✅ RDMA library: Loaded")
	}

	if !requirements["rdma_devices_configured"] {
		errors = append(errors, "RDMA devices are not configured")
	} else {
		fmt.Println("✅ RDMA devices: Configured")
	}

	if !requirements["iommu_disabled"] {
		errors = append(errors, "IOMMU is not disabled (should be disabled for optimal GDS performance)")
	} else {
		fmt.Println("✅ IOMMU: Disabled")
	}

	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("GDS validation failed:\n  • %s", strings.Join(errors, "\n  • "))
	}

	return nil
}

func runHostStatus(verbose bool) error {
	fmt.Println("📊 AMG Environment Status")
	fmt.Println("=" + strings.Repeat("=", 50))

	// Check UV virtual environment status
	if err := showUvEnvironmentStatus(); err != nil {
		fmt.Printf("❌ Error checking UV environment: %v\n", err)
	}

	fmt.Println() // Add spacing

	// Check repository status
	if err := showRepositoryStatus(); err != nil {
		fmt.Printf("❌ Error checking repository: %v\n", err)
	}

	// Show PyTorch configuration, installed packages and system resources only in verbose mode
	if verbose {
		fmt.Println() // Add spacing

		// Show PyTorch configuration
		if err := showPyTorchInfo(); err != nil {
			fmt.Printf("❌ Error checking PyTorch configuration: %v\n", err)
		}

		fmt.Println() // Add spacing

		// Show installed packages
		if err := showInstalledPackages(); err != nil {
			fmt.Printf("❌ Error checking installed packages: %v\n", err)
		}

		fmt.Println() // Add spacing

		// Show system resources
		if err := showSystemResources(); err != nil {
			fmt.Printf("❌ Error checking system resources: %v\n", err)
		}
	} else {
		fmt.Println()
		fmt.Println("💡 Use --verbose or -v to show PyTorch configuration, installed packages and system resources")
	}

	return nil
}

func runHostClear(cmd *cobra.Command) error {
	fmt.Println("🧹 Clearing AMG environment...")
	fmt.Println()

	// Check if --yes flag was provided
	skipConfirmation, _ := cmd.Flags().GetBool("yes")

	// Show what will be deleted
	basePath := getBasePath()
	fmt.Printf("This will permanently delete the AMG environment directory and all its contents:\n")
	fmt.Printf("  📁 %s\n", basePath)
	fmt.Println("  - UV virtual environment (.venv)")
	fmt.Println("  - LMCache repository")
	fmt.Println("  - All installed packages")
	fmt.Println("  - Setup state and configuration")
	fmt.Println()

	// Ask for confirmation unless --yes flag was provided
	if !skipConfirmation {
		confirmed, err := askForConfirmation("Are you sure you want to proceed with this destructive operation?")
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		if !confirmed {
			fmt.Println("❌ Operation cancelled by user")
			return nil
		}

		fmt.Println("✅ Confirmed. Proceeding with cleanup...")
	} else {
		fmt.Println("✅ Skipping confirmation (--yes flag provided). Proceeding with cleanup...")
	}
	fmt.Println()

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

// showPyTorchInfo displays PyTorch version and supported backends for vLLM
func showPyTorchInfo() error {
	fmt.Println("🔥 PyTorch Configuration:")
	fmt.Println("-" + strings.Repeat("-", 24))

	basePath := getBasePath()

	// Create a Python script to check PyTorch configuration
	pythonScript := `
import sys
try:
    import torch
    print(f"PyTorch Version: {torch.__version__}")
    
    # Check CUDA support
    if torch.cuda.is_available():
        cuda_version = torch.version.cuda if hasattr(torch.version, 'cuda') else "unknown"
        device_count = torch.cuda.device_count()
        device_name = torch.cuda.get_device_name(0) if device_count > 0 else "unknown"
        print(f"CUDA Available: Yes (version {cuda_version})")
        print(f"CUDA Devices: {device_count}")
        if device_count > 0:
            print(f"Primary GPU: {device_name}")
    else:
        print("CUDA Available: No")
    
    # Check cuDNN support
    if hasattr(torch.backends, 'cudnn') and torch.backends.cudnn.enabled:
        print("cuDNN: Enabled")
    else:
        print("cuDNN: Disabled/Not available")
    
    # Check ROCm support (AMD GPUs)
    if hasattr(torch.version, 'hip') and torch.version.hip is not None:
        print(f"ROCm/HIP Available: Yes (version {torch.version.hip})")
    else:
        print("ROCm/HIP Available: No")
    
    # Check MPS support (Apple Silicon)
    if hasattr(torch.backends, 'mps') and torch.backends.mps.is_available():
        print("MPS (Apple Silicon): Available")
    else:
        print("MPS (Apple Silicon): Not available")
    
    # Check CPU-only mode
    print(f"CPU Threads: {torch.get_num_threads()}")
    
except ImportError as e:
    print(f"Error: PyTorch not available - {e}")
    sys.exit(1)
except Exception as e:
    print(f"Error checking PyTorch configuration: {e}")
    sys.exit(1)
`

	// Execute the Python script using uv run
	cmd := exec.Command("uv", "run", "python", "-c", pythonScript)
	cmd.Dir = basePath
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ Could not retrieve PyTorch configuration: %v\n", err)
		return nil
	}

	// Display the output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Error:") {
			fmt.Printf("❌ %s\n", line)
		} else if strings.Contains(line, "Available: Yes") || strings.Contains(line, "Enabled") {
			fmt.Printf("✅ %s\n", line)
		} else if strings.Contains(line, "Available: No") || strings.Contains(line, "Disabled") || strings.Contains(line, "Not available") {
			fmt.Printf("❌ %s\n", line)
		} else {
			fmt.Printf("ℹ️  %s\n", line)
		}
	}

	return nil
}

// showInstalledPackages displays information about installed Python packages
func showInstalledPackages() error {
	fmt.Println("📦 Installed Packages:")
	fmt.Println("-" + strings.Repeat("-", 20))

	basePath := getBasePath()

	// Check key packages that should be installed
	keyPackages := []string{
		"vllm",
		"torch",
		"transformers",
		"py-spy",
		"scalene",
		"pyinstrument",
		"line_profiler",
		"fastsafetensors",
	}

	fmt.Println("🔍 Checking key packages:")
	for _, pkg := range keyPackages {
		cmd := exec.Command("uv", "pip", "show", pkg)
		cmd.Dir = basePath
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("❌ %s: Not installed\n", pkg)
		} else {
			// Extract version from pip show output
			lines := strings.Split(string(output), "\n")
			version := "unknown"
			for _, line := range lines {
				if strings.HasPrefix(line, "Version: ") {
					version = strings.TrimPrefix(line, "Version: ")
					break
				}
			}
			fmt.Printf("✅ %s: %s\n", pkg, version)
		}
	}

	// Show total package count
	cmd := exec.Command("uv", "pip", "list", "--format=freeze")
	cmd.Dir = basePath
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("⚠️  Could not list all packages: %v\n", err)
	} else {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		packageCount := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				packageCount++
			}
		}
		fmt.Printf("📊 Total packages installed: %d\n", packageCount)
	}

	return nil
}

// showSystemResources displays system resource information
func showSystemResources() error {
	fmt.Println("💻 System Resources:")
	fmt.Println("-" + strings.Repeat("-", 18))

	// Operating system info
	fmt.Printf("🖥️  Operating System: %s %s\n", runtime.GOOS, runtime.GOARCH)

	// CPU info
	fmt.Printf("⚙️  CPU Cores: %d\n", runtime.NumCPU())

	// Memory info (Linux specific)
	if runtime.GOOS == "linux" {
		showLinuxMemoryInfo()
	}

	// Disk space for AMG directory
	showDiskSpace()

	// GPU info (if available)
	showGPUInfo()

	return nil
}

// showLinuxMemoryInfo displays Linux-specific memory information
func showLinuxMemoryInfo() {
	// Read /proc/meminfo for memory information
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		fmt.Printf("⚠️  Could not read memory info: %v\n", err)
		return
	}

	lines := strings.Split(string(data), "\n")
	memInfo := make(map[string]string)

	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				memInfo[key] = value
			}
		}
	}

	if memTotal, ok := memInfo["MemTotal"]; ok {
		fmt.Printf("🧠 Memory Total: %s\n", memTotal)
	}
	if memAvailable, ok := memInfo["MemAvailable"]; ok {
		fmt.Printf("🧠 Memory Available: %s\n", memAvailable)
	}
}

// showDiskSpace displays disk space information for the AMG directory
func showDiskSpace() {
	basePath := getBasePath()

	// Use df command to get disk space
	cmd := exec.Command("df", "-h", basePath)
	output, err := cmd.Output()
	if err != nil {
		// If basePath doesn't exist, check parent directory
		parentPath := filepath.Dir(basePath)
		cmd = exec.Command("df", "-h", parentPath)
		output, err = cmd.Output()
		if err != nil {
			fmt.Printf("⚠️  Could not check disk space: %v\n", err)
			return
		}
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) >= 2 {
		// Parse df output (typically: Filesystem Size Used Avail Use% Mounted on)
		fields := strings.Fields(lines[1])
		if len(fields) >= 4 {
			fmt.Printf("💾 Disk Available: %s (Total: %s, Used: %s)\n", fields[3], fields[1], fields[2])
		}
	}
}

// showGPUInfo displays GPU information if available
func showGPUInfo() {
	// Try nvidia-smi first
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,memory.used,memory.free", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		fmt.Printf("🎮 GPU Count: %d\n", len(lines))
		for i, line := range lines {
			fields := strings.Split(line, ", ")
			if len(fields) >= 4 {
				name := fields[0]
				total := fields[1]
				used := fields[2]
				free := fields[3]
				fmt.Printf("   GPU %d: %s (Memory: %s MB total, %s MB used, %s MB free)\n",
					i, name, total, used, free)
			}
		}
		return
	}

	// Try lspci for basic GPU info
	cmd = exec.Command("lspci")
	output, err = cmd.Output()
	if err != nil {
		fmt.Println("ℹ️  No GPU information available")
		return
	}

	lines := strings.Split(string(output), "\n")
	gpuCount := 0
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "vga") ||
			strings.Contains(strings.ToLower(line), "3d") ||
			strings.Contains(strings.ToLower(line), "display") {
			if gpuCount == 0 {
				fmt.Println("🎮 GPU Devices:")
			}
			gpuCount++
			// Extract device name (after colon)
			if idx := strings.Index(line, ": "); idx != -1 {
				deviceName := line[idx+2:]
				fmt.Printf("   GPU %d: %s\n", gpuCount-1, deviceName)
			}
		}
	}

	if gpuCount == 0 {
		fmt.Println("ℹ️  No GPU devices found")
	}
}

// showUvEnvironmentStatus displays the status of the UV virtual environment
func showUvEnvironmentStatus() error {
	fmt.Println("🐍 UV Virtual Environment Status:")
	fmt.Println("-" + strings.Repeat("-", 30))

	basePath := getBasePath()
	uvEnvPath := getUvEnvPath()

	// Check if base directory exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		fmt.Println("❌ AMG environment directory not found")
		fmt.Printf("   Expected location: %s\n", basePath)
		fmt.Println("   Run 'amgctl host setup' to create the environment")
		return nil
	}

	fmt.Printf("✅ Base directory: %s\n", basePath)

	// Check if UV virtual environment exists
	if _, err := os.Stat(uvEnvPath); os.IsNotExist(err) {
		fmt.Println("❌ UV virtual environment not found")
		fmt.Printf("   Expected location: %s\n", uvEnvPath)
		fmt.Println("   Run 'amgctl host setup' to create the environment")
		return nil
	}

	fmt.Printf("✅ UV virtual environment: %s\n", uvEnvPath)

	// Check if UV command is available
	if !commandExists("uv") {
		fmt.Println("❌ UV command not found in PATH")
		return nil
	}

	// Check Python version in the virtual environment
	cmd := exec.Command("uv", "run", "python", "--version")
	cmd.Dir = basePath
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("⚠️  Could not determine Python version: %v\n", err)
	} else {
		pythonVersion := strings.TrimSpace(string(output))
		fmt.Printf("✅ Python version: %s\n", pythonVersion)
	}

	// Check if environment is currently active
	virtualEnv := os.Getenv("VIRTUAL_ENV")
	if virtualEnv == uvEnvPath {
		fmt.Println("✅ Virtual environment is currently ACTIVE")
	} else if virtualEnv != "" {
		fmt.Printf("⚠️  Different virtual environment is active: %s\n", virtualEnv)
	} else {
		fmt.Println("ℹ️  Virtual environment is not currently active")
		fmt.Println("   To activate: source " + filepath.Join(uvEnvPath, "bin", "activate"))
	}

	return nil
}

// showRepositoryStatus displays the status of the LMCache repository
func showRepositoryStatus() error {
	fmt.Println("📁 Repository Status:")
	fmt.Println("-" + strings.Repeat("-", 20))

	repoPath := getRepoPath()

	// Check if repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		fmt.Println("❌ LMCache repository not found")
		fmt.Printf("   Expected location: %s\n", repoPath)
		fmt.Println("   Run 'amgctl host setup' to clone the repository")
		return nil
	}

	fmt.Printf("✅ Repository location: %s\n", repoPath)

	// Load setup state to see configuration
	state, err := loadSetupState()
	if err != nil {
		fmt.Printf("⚠️  Could not load setup state: %v\n", err)
	} else if state != nil {
		fmt.Printf("📋 Repository URL: %s\n", state.LMCacheRepo)
		if state.LMCacheBranch != "" {
			fmt.Printf("🌿 Following branch: %s\n", state.LMCacheBranch)
		} else if state.LMCacheCommit != "" {
			fmt.Printf("📌 Pinned to commit: %s\n", state.LMCacheCommit)
		}
	}

	// Get current commit
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("⚠️  Could not get current commit: %v\n", err)
	} else {
		currentCommit := strings.TrimSpace(string(output))
		fmt.Printf("📍 Current commit: %s\n", currentCommit[:8]+"...")
	}

	// Get current branch/status
	cmd = exec.Command("git", "-C", repoPath, "branch", "--show-current")
	output, err = cmd.Output()
	if err == nil {
		currentBranch := strings.TrimSpace(string(output))
		if currentBranch != "" {
			fmt.Printf("🌿 Current branch: %s\n", currentBranch)
		} else {
			fmt.Println("📍 In detached HEAD state")
		}
	}

	// Check for uncommitted changes
	cmd = exec.Command("git", "-C", repoPath, "status", "--porcelain")
	output, err = cmd.Output()
	if err != nil {
		fmt.Printf("⚠️  Could not check git status: %v\n", err)
	} else {
		changes := strings.TrimSpace(string(output))
		if changes == "" {
			fmt.Println("✅ Working directory is clean")
		} else {
			fmt.Println("⚠️  Uncommitted changes detected:")
			lines := strings.Split(changes, "\n")
			for i, line := range lines {
				if i < 5 { // Show first 5 changes
					fmt.Printf("   %s\n", line)
				} else {
					fmt.Printf("   ... and %d more changes\n", len(lines)-5)
					break
				}
			}
		}
	}

	return nil
}
