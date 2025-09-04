package scheduler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Utility functions for amgctl operations

// downloadAmgctlBinary downloads the latest amgctl binary from GitHub
func downloadAmgctlBinary(filepath string, logs *strings.Builder) error {
	binaryURL := "https://github.com/weka/amg-utils/releases/latest/download/amgctl-linux-amd64"
	fmt.Fprintf(logs, "Downloading amgctl binary from: %s\n", binaryURL)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(binaryURL)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("binary download returned status: %s", resp.Status)
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create binary file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save binary: %w", err)
	}

	fmt.Fprintf(logs, "Binary downloaded to: %s\n", filepath)
	return nil
}

// getAmgctlVersion runs the --version command and extracts the version
func getAmgctlVersion(binaryPath, workingDir string, logs *strings.Builder) (string, error) {
	fmt.Fprintf(logs, "Running: %s --version\n", binaryPath)

	cmd := exec.Command(binaryPath, "--version")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run version command: %w", err)
	}

	versionOutput := strings.TrimSpace(string(output))
	fmt.Fprintf(logs, "Version command output: %s\n", versionOutput)

	// Extract version from output like "amgctl version 0.1.16"
	parts := strings.Fields(versionOutput)
	if len(parts) < 3 {
		return "", fmt.Errorf("unexpected version output format: %s", versionOutput)
	}

	version := parts[2] // Get the version number
	fmt.Fprintf(logs, "Extracted version: %s\n", version)
	return version, nil
}

// runAmgctlCommand executes an amgctl command with logging and error handling
func runAmgctlCommand(binaryPath, workingDir string, args []string, logs *strings.Builder) error {
	commandStr := strings.Join(args, " ")
	fmt.Fprintf(logs, "Running amgctl %s command...\n", commandStr)

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workingDir

	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	fmt.Fprintf(logs, "Command 'amgctl %s' output:\n%s\n", commandStr, outputStr)

	if err != nil {
		return fmt.Errorf("command 'amgctl %s' failed: %w", commandStr, err)
	}

	fmt.Fprintf(logs, "✅ Command 'amgctl %s' completed successfully\n", commandStr)
	return nil
}

// setupAmgctlBinary downloads and sets up the amgctl binary, returning the binary path
func setupAmgctlBinary(tempDir string, logs *strings.Builder) (string, error) {
	binaryPath := filepath.Join(tempDir, "amgctl-linux-amd64")

	if err := downloadAmgctlBinary(binaryPath, logs); err != nil {
		return "", fmt.Errorf("failed to download amgctl: %w", err)
	}

	if err := os.Chmod(binaryPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}
	logs.WriteString("Made binary executable\n")

	return binaryPath, nil
}

// AmgctlFetchLatestTest validates the amgctl binary from GitHub
type AmgctlFetchLatestTest struct {
	Name            string
	ExpectedVersion string
	TempDir         string
}

// AmgctlUpgradeToLatestTest tests the upgrade functionality of amgctl
type AmgctlUpgradeToLatestTest struct {
	Name           string
	CurrentVersion string
	OlderVersion   string
	TempDir        string
}

// AmgctlSetupTest tests the host setup functionality of amgctl
type AmgctlSetupTest struct {
	Name    string
	TempDir string
}

// AmgctlOnDiagnosticsTest tests various diagnostic commands of amgctl
type AmgctlOnDiagnosticsTest struct {
	Name         string
	Dependencies []string
	TempDir      string
}

// NewAmgctlFetchLatestTest creates a new amgctl validation test
func NewAmgctlFetchLatestTest(expectedVersion string) *AmgctlFetchLatestTest {
	return &AmgctlFetchLatestTest{
		Name:            "amgctl_fetch_latest_test",
		ExpectedVersion: expectedVersion,
	}
}

// NewAmgctlUpgradeToLatestTest creates a new amgctl upgrade test
func NewAmgctlUpgradeToLatestTest(currentVersion string) *AmgctlUpgradeToLatestTest {
	return &AmgctlUpgradeToLatestTest{
		Name:           "amgctl_upgrade_to_latest_test",
		CurrentVersion: currentVersion,
		OlderVersion:   "0.1.14", // Known stable older version for testing
	}
}

// NewAmgctlSetupTest creates a new amgctl setup test
func NewAmgctlSetupTest() *AmgctlSetupTest {
	return &AmgctlSetupTest{
		Name: "amgctl_setup_test",
	}
}

// NewAmgctlOnDiagnosticsTest creates a new diagnostic test that depends on AmgctlSetupTest
func NewAmgctlOnDiagnosticsTest() *AmgctlOnDiagnosticsTest {
	return &AmgctlOnDiagnosticsTest{
		Name:         "amgctl_on_diagnostics_test",
		Dependencies: []string{"amgctl_setup_test"},
	}
}

// GetName returns the test name
func (t *AmgctlFetchLatestTest) GetName() string {
	return t.Name
}

// GetName returns the test name
func (t *AmgctlUpgradeToLatestTest) GetName() string {
	return t.Name
}

// GetName returns the test name
func (t *AmgctlSetupTest) GetName() string {
	return t.Name
}

// GetName returns the test name
func (t *AmgctlOnDiagnosticsTest) GetName() string {
	return t.Name
}

// GetDependencies returns the list of tests this test depends on
func (t *AmgctlOnDiagnosticsTest) GetDependencies() []string {
	return t.Dependencies
}

// RunTest downloads amgctl from GitHub and validates its version
func (t *AmgctlFetchLatestTest) RunTest() (bool, time.Duration, string, error) {
	start := time.Now()
	var logs strings.Builder

	logs.WriteString(fmt.Sprintf("Starting test: %s\n", t.Name))
	logs.WriteString(fmt.Sprintf("Expected version: %s\n", t.ExpectedVersion))

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "amg-qad-test-")
	if err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to create temp directory: %v\n", err))
		return false, duration, logs.String(), err
	}
	t.TempDir = tempDir
	defer func() {
		if cleanupErr := os.RemoveAll(tempDir); cleanupErr != nil {
			logs.WriteString(fmt.Sprintf("WARNING: Failed to cleanup temp directory: %v\n", cleanupErr))
		}
	}()

	logs.WriteString(fmt.Sprintf("Using temp directory: %s\n", tempDir))

	// Setup amgctl binary
	binaryPath, err := setupAmgctlBinary(tempDir, &logs)
	if err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to setup amgctl: %v\n", err))
		return false, duration, logs.String(), err
	}

	// Run version check
	version, err := getAmgctlVersion(binaryPath, t.TempDir, &logs)
	if err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to get version: %v\n", err))
		return false, duration, logs.String(), err
	}

	// Validate version
	passed := version == t.ExpectedVersion
	logs.WriteString(fmt.Sprintf("Validating version: actual='%s', expected='%s'\n", version, t.ExpectedVersion))
	duration := time.Since(start)

	if passed {
		logs.WriteString(fmt.Sprintf("SUCCESS: Version validation passed (got: %s, expected: %s)\n", version, t.ExpectedVersion))
	} else {
		logs.WriteString(fmt.Sprintf("FAILED: Version mismatch (got: %s, expected: %s)\n", version, t.ExpectedVersion))
	}

	logs.WriteString(fmt.Sprintf("Test duration: %v\n", duration))

	return passed, duration, logs.String(), nil
}

// RunTest downloads older amgctl, tests upgrade functionality, and validates the result
func (t *AmgctlUpgradeToLatestTest) RunTest() (bool, time.Duration, string, error) {
	start := time.Now()
	var logs strings.Builder

	logs.WriteString(fmt.Sprintf("Starting test: %s\n", t.Name))
	logs.WriteString(fmt.Sprintf("Testing upgrade from %s to %s\n", t.OlderVersion, t.CurrentVersion))

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "amg-qad-upgrade-test-")
	if err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to create temp directory: %v\n", err))
		return false, duration, logs.String(), err
	}
	t.TempDir = tempDir
	defer func() {
		if cleanupErr := os.RemoveAll(tempDir); cleanupErr != nil {
			logs.WriteString(fmt.Sprintf("WARNING: Failed to cleanup temp directory: %v\n", cleanupErr))
		}
	}()

	logs.WriteString(fmt.Sprintf("Using temp directory: %s\n", tempDir))

	// Step 1: Download older version
	binaryPath := filepath.Join(tempDir, "amgctl-linux-amd64")
	if err := t.downloadOlderAmgctl(binaryPath, &logs); err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to download older amgctl: %v\n", err))
		return false, duration, logs.String(), err
	}

	// Make binary executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to make binary executable: %v\n", err))
		return false, duration, logs.String(), err
	}
	logs.WriteString("Made binary executable\n")

	// Step 2: Verify initial version is older
	initialVersion, err := getAmgctlVersion(binaryPath, t.TempDir, &logs)
	if err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to get initial version: %v\n", err))
		return false, duration, logs.String(), err
	}

	if initialVersion != t.OlderVersion {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Initial version mismatch (got: %s, expected: %s)\n", initialVersion, t.OlderVersion))
		return false, duration, logs.String(), fmt.Errorf("initial version mismatch")
	}
	logs.WriteString(fmt.Sprintf("✅ Initial version verified: %s\n", initialVersion))

	// Step 3: Run upgrade command
	if err := t.runUpgradeCommand(binaryPath, &logs); err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Upgrade command failed: %v\n", err))
		return false, duration, logs.String(), err
	}

	// Step 4: Verify final version is current
	finalVersion, err := getAmgctlVersion(binaryPath, t.TempDir, &logs)
	if err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to get final version: %v\n", err))
		return false, duration, logs.String(), err
	}

	// Validate upgrade success
	passed := finalVersion == t.CurrentVersion
	duration := time.Since(start)

	if passed {
		logs.WriteString(fmt.Sprintf("SUCCESS: Upgrade completed successfully (from %s to %s)\n", initialVersion, finalVersion))
	} else {
		logs.WriteString(fmt.Sprintf("FAILED: Upgrade unsuccessful (expected: %s, got: %s)\n", t.CurrentVersion, finalVersion))
	}

	logs.WriteString(fmt.Sprintf("Test duration: %v\n", duration))
	return passed, duration, logs.String(), nil
}

// downloadOlderAmgctl downloads the specified older version of amgctl
func (t *AmgctlUpgradeToLatestTest) downloadOlderAmgctl(filepath string, logs *strings.Builder) error {
	url := fmt.Sprintf("https://github.com/weka/amg-utils/releases/download/v%s/amgctl-linux-amd64", t.OlderVersion)
	fmt.Fprintf(logs, "Downloading older amgctl from: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download from %s: %w", url, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(logs, "WARNING: Failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(logs, "WARNING: Failed to close file: %v\n", closeErr)
		}
	}()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	logs.WriteString("✅ Older amgctl binary downloaded successfully\n")
	return nil
}

// runUpgradeCommand executes the update command
func (t *AmgctlUpgradeToLatestTest) runUpgradeCommand(binaryPath string, logs *strings.Builder) error {
	logs.WriteString("Running upgrade command...\n")

	if err := runAmgctlCommand(binaryPath, t.TempDir, []string{"update"}, logs); err != nil {
		return fmt.Errorf("upgrade command failed: %w", err)
	}

	logs.WriteString("✅ Upgrade command completed successfully\n")
	return nil
}

// RunTest downloads amgctl and tests the host setup functionality
func (t *AmgctlSetupTest) RunTest() (bool, time.Duration, string, error) {
	start := time.Now()
	var logs strings.Builder

	logs.WriteString(fmt.Sprintf("Starting test: %s\n", t.Name))
	logs.WriteString("Testing amgctl host clear and setup commands\n")

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "amg-qad-setup-test-")
	if err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to create temp directory: %v\n", err))
		return false, duration, logs.String(), err
	}
	t.TempDir = tempDir
	defer func() {
		if cleanupErr := os.RemoveAll(tempDir); cleanupErr != nil {
			logs.WriteString(fmt.Sprintf("WARNING: Failed to cleanup temp directory: %v\n", cleanupErr))
		}
	}()

	logs.WriteString(fmt.Sprintf("Using temp directory: %s\n", tempDir))

	// Setup amgctl binary
	binaryPath, err := setupAmgctlBinary(tempDir, &logs)
	if err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to setup amgctl: %v\n", err))
		return false, duration, logs.String(), err
	}

	// Step 1: Run amgctl host clear
	if err := runAmgctlCommand(binaryPath, t.TempDir, []string{"host", "clear", "--yes"}, &logs); err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Host clear command failed: %v\n", err))
		return false, duration, logs.String(), err
	}

	// Step 2: Run amgctl host setup
	if err := runAmgctlCommand(binaryPath, t.TempDir, []string{"host", "setup"}, &logs); err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Host setup command failed: %v\n", err))
		return false, duration, logs.String(), err
	}

	duration := time.Since(start)
	logs.WriteString("SUCCESS: Both host clear and setup commands completed successfully\n")
	logs.WriteString(fmt.Sprintf("Test duration: %v\n", duration))

	return true, duration, logs.String(), nil
}

// RunTest runs the diagnostic test commands
func (t *AmgctlOnDiagnosticsTest) RunTest() (bool, time.Duration, string, error) {
	start := time.Now()
	var logs strings.Builder

	logs.WriteString(fmt.Sprintf("Starting test: %s\n", t.Name))
	logs.WriteString("Testing amgctl diagnostic commands\n")
	logs.WriteString("Dependencies: " + fmt.Sprintf("%v", t.Dependencies) + "\n")

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "amg-qad-diagnostics-test-")
	if err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to create temp directory: %v\n", err))
		return false, duration, logs.String(), err
	}
	t.TempDir = tempDir
	defer func() {
		if cleanupErr := os.RemoveAll(tempDir); cleanupErr != nil {
			logs.WriteString(fmt.Sprintf("WARNING: Failed to cleanup temp directory: %v\n", cleanupErr))
		}
	}()

	logs.WriteString(fmt.Sprintf("Using temp directory: %s\n", tempDir))

	// Setup amgctl binary
	binaryPath, err := setupAmgctlBinary(tempDir, &logs)
	if err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to setup amgctl: %v\n", err))
		return false, duration, logs.String(), err
	}

	// Run host setup first (since we depend on AmgctlSetupTest)
	if err := runAmgctlCommand(binaryPath, t.TempDir, []string{"host", "setup"}, &logs); err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Host setup command failed: %v\n", err))
		return false, duration, logs.String(), err
	}

	// Define all diagnostic commands to run
	diagnosticCommands := [][]string{
		{"host", "status"},
		{"host", "status", "-v"},
		{"host", "pre-flight"},
		{"host", "pre-flight", "--full"},
		{"hw", "show"},
		{"hw", "net"},
	}

	// Run all diagnostic commands
	allPassed := true
	for _, cmdArgs := range diagnosticCommands {
		if err := runAmgctlCommand(binaryPath, t.TempDir, cmdArgs, &logs); err != nil {
			logs.WriteString(fmt.Sprintf("WARNING: Command '%s %s' had issues: %v\n", binaryPath, strings.Join(cmdArgs, " "), err))
			// Don't fail the test for diagnostic command warnings - just log them
		}
	}

	duration := time.Since(start)
	if allPassed {
		logs.WriteString("SUCCESS: All diagnostic commands completed\n")
	} else {
		logs.WriteString("COMPLETED: Some diagnostic commands had warnings, but test finished\n")
	}
	logs.WriteString(fmt.Sprintf("Test duration: %v\n", duration))

	return allPassed, duration, logs.String(), nil
}
