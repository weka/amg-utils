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

// GetName returns the test name
func (t *AmgctlFetchLatestTest) GetName() string {
	return t.Name
}

// GetName returns the test name
func (t *AmgctlUpgradeToLatestTest) GetName() string {
	return t.Name
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

	// Download amgctl binary
	binaryPath := filepath.Join(tempDir, "amgctl-linux-amd64")
	if err := t.downloadAmgctl(binaryPath, &logs); err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to download amgctl: %v\n", err))
		return false, duration, logs.String(), err
	}

	// Make binary executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to make binary executable: %v\n", err))
		return false, duration, logs.String(), err
	}
	logs.WriteString("Made binary executable\n")

	// Run version check
	version, err := t.getVersion(binaryPath, &logs)
	if err != nil {
		duration := time.Since(start)
		logs.WriteString(fmt.Sprintf("ERROR: Failed to get version: %v\n", err))
		return false, duration, logs.String(), err
	}

	// Validate version
	passed := t.validateVersion(version, &logs)
	duration := time.Since(start)

	if passed {
		logs.WriteString(fmt.Sprintf("SUCCESS: Version validation passed (got: %s, expected: %s)\n", version, t.ExpectedVersion))
	} else {
		logs.WriteString(fmt.Sprintf("FAILED: Version mismatch (got: %s, expected: %s)\n", version, t.ExpectedVersion))
	}

	logs.WriteString(fmt.Sprintf("Test duration: %v\n", duration))

	return passed, duration, logs.String(), nil
}

// downloadAmgctl downloads the latest amgctl binary from GitHub
func (t *AmgctlFetchLatestTest) downloadAmgctl(filepath string, logs *strings.Builder) error {
	// Direct URL to latest amgctl binary (much simpler!)
	binaryURL := "https://github.com/weka/amg-utils/releases/latest/download/amgctl-linux-amd64"

	fmt.Fprintf(logs, "Downloading amgctl binary from: %s\n", binaryURL)

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 30 * time.Second}

	// Download the binary directly
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

	// Save binary to file
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

// getVersion runs the binary and extracts its version
func (t *AmgctlFetchLatestTest) getVersion(binaryPath string, logs *strings.Builder) (string, error) {
	fmt.Fprintf(logs, "Running: %s --version\n", binaryPath)

	cmd := exec.Command(binaryPath, "--version")
	cmd.Dir = t.TempDir

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

// validateVersion checks if the actual version matches expected
func (t *AmgctlFetchLatestTest) validateVersion(actualVersion string, logs *strings.Builder) bool {
	fmt.Fprintf(logs, "Validating version: actual='%s', expected='%s'\n", actualVersion, t.ExpectedVersion)

	return actualVersion == t.ExpectedVersion
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
	initialVersion, err := t.getAmgctlVersion(binaryPath, &logs)
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
	finalVersion, err := t.getAmgctlVersion(binaryPath, &logs)
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

// getAmgctlVersion runs the --version command and extracts the version
func (t *AmgctlUpgradeToLatestTest) getAmgctlVersion(binaryPath string, logs *strings.Builder) (string, error) {
	logs.WriteString("Getting amgctl version...\n")

	cmd := exec.Command(binaryPath, "--version")
	cmd.Dir = t.TempDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run version command: %w", err)
	}

	versionOutput := strings.TrimSpace(string(output))
	fmt.Fprintf(logs, "Version command output: %s\n", versionOutput)

	// Extract version from output like "amgctl version 0.1.14"
	parts := strings.Fields(versionOutput)
	if len(parts) < 3 || parts[0] != "amgctl" || parts[1] != "version" {
		return "", fmt.Errorf("unexpected version output format: %s", versionOutput)
	}

	version := parts[2]
	fmt.Fprintf(logs, "Extracted version: %s\n", version)

	return version, nil
}

// runUpgradeCommand executes the update command
func (t *AmgctlUpgradeToLatestTest) runUpgradeCommand(binaryPath string, logs *strings.Builder) error {
	logs.WriteString("Running upgrade command...\n")

	cmd := exec.Command(binaryPath, "update")
	cmd.Dir = t.TempDir

	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	fmt.Fprintf(logs, "Upgrade command output:\n%s\n", outputStr)

	if err != nil {
		return fmt.Errorf("upgrade command failed: %w", err)
	}

	// Check if the output indicates success
	if strings.Contains(outputStr, "updated successfully") || strings.Contains(outputStr, "✅") {
		logs.WriteString("✅ Upgrade command completed successfully\n")
		return nil
	}

	logs.WriteString("⚠️  Upgrade command completed but success indicators not found\n")
	return nil
}
