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
		Name:            "AMGctl GitHub Version Validation",
		ExpectedVersion: "0.1.16",
	}
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
	// GitHub releases API URL for latest release
	releaseURL := "https://api.github.com/repos/weka/amg-utils/releases/latest"

	fmt.Fprintf(logs, "Fetching latest release info from: %s\n", releaseURL)

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 30 * time.Second}

	// Get latest release info
	resp, err := client.Get(releaseURL)
	if err != nil {
		return fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	// Parse JSON to find amgctl-linux-amd64 asset
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Simple JSON parsing to find the binary download URL
	bodyStr := string(body)
	binaryURL, found := t.extractBinaryURL(bodyStr, logs)
	if !found {
		return fmt.Errorf("amgctl-linux-amd64 binary not found in latest release")
	}

	fmt.Fprintf(logs, "Downloading binary from: %s\n", binaryURL)

	// Download the binary
	binResp, err := client.Get(binaryURL)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer func() {
		_ = binResp.Body.Close()
	}()

	if binResp.StatusCode != http.StatusOK {
		return fmt.Errorf("binary download returned status: %s", binResp.Status)
	}

	// Save binary to file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create binary file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	_, err = io.Copy(file, binResp.Body)
	if err != nil {
		return fmt.Errorf("failed to save binary: %w", err)
	}

	fmt.Fprintf(logs, "Binary downloaded to: %s\n", filepath)
	return nil
}

// extractBinaryURL extracts the download URL for amgctl-linux-amd64 from GitHub API response
func (t *AmgctlTest) extractBinaryURL(jsonBody string, logs *strings.Builder) (string, bool) {
	// Simple JSON parsing - look for amgctl-linux-amd64 asset
	// Look for all download URLs that contain amgctl-linux-amd64

	// Split by lines and look for browser_download_url lines that contain our binary
	lines := strings.Split(jsonBody, "\n")

	for _, line := range lines {
		// Look for browser_download_url lines
		if strings.Contains(line, "browser_download_url") {
			// Extract the URL
			start := strings.Index(line, "https://")
			if start == -1 {
				continue
			}
			end := strings.LastIndex(line, `"`)
			if end == -1 || end <= start {
				continue
			}

			url := line[start:end]

			// Check if this URL is for our binary
			if strings.Contains(url, "amgctl-linux-amd64") {
				fmt.Fprintf(logs, "Found binary URL: %s\n", url)
				return url, true
			}
		}
	}

	// Also try a more direct approach - search for the binary name and then find the next URL
	for i, line := range lines {
		if strings.Contains(line, "amgctl-linux-amd64") {
			// Look for browser_download_url in the next few lines
			for j := i; j < len(lines) && j < i+10; j++ {
				if strings.Contains(lines[j], "browser_download_url") {
					start := strings.Index(lines[j], "https://")
					if start == -1 {
						continue
					}
					end := strings.LastIndex(lines[j], `"`)
					if end == -1 || end <= start {
						continue
					}

					url := lines[j][start:end]
					fmt.Fprintf(logs, "Found binary URL (fallback method): %s\n", url)
					return url, true
				}
			}
		}
	}

	// Debug: log some info about what we found
	assetCount := strings.Count(jsonBody, "browser_download_url")
	fmt.Fprintf(logs, "DEBUG: Found %d total download URLs in release\n", assetCount)

	// Log the first few URLs for debugging
	urlCount := 0
	for _, line := range lines {
		if strings.Contains(line, "browser_download_url") && urlCount < 3 {
			start := strings.Index(line, "https://")
			end := strings.LastIndex(line, `"`)
			if start != -1 && end > start {
				url := line[start:end]
				fmt.Fprintf(logs, "DEBUG: Available URL %d: %s\n", urlCount+1, url)
				urlCount++
			}
		}
	}

	return "", false
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
