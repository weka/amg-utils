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

// AmgctlTest validates the amgctl binary from GitHub
type AmgctlTest struct {
	Name            string
	ExpectedVersion string
	TempDir         string
}

// NewAmgctlTest creates a new amgctl validation test
func NewAmgctlTest() *AmgctlTest {
	return &AmgctlTest{
		Name:            "amgctl_integration_test",
		ExpectedVersion: "0.1.16",
	}
}

// GetName returns the test name
func (t *AmgctlTest) GetName() string {
	return t.Name
}

// RunTest downloads amgctl from GitHub and validates its version
func (t *AmgctlTest) RunTest() (bool, time.Duration, string, error) {
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
func (t *AmgctlTest) downloadAmgctl(filepath string, logs *strings.Builder) error {
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
func (t *AmgctlTest) getVersion(binaryPath string, logs *strings.Builder) (string, error) {
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
func (t *AmgctlTest) validateVersion(actualVersion string, logs *strings.Builder) bool {
	fmt.Fprintf(logs, "Validating version: actual='%s', expected='%s'\n", actualVersion, t.ExpectedVersion)

	return actualVersion == t.ExpectedVersion
}
