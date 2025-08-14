package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	repoOwner      = "weka"
	updateRepoName = "amg-utils"
	githubAPIURL   = "https://api.github.com"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update amgctl to the latest version",
	Long: `Update amgctl to the latest version from GitHub releases.
This command will:
  - Check for the latest release on GitHub
  - Download the appropriate binary for your platform
  - Verify the download integrity
  - Replace the current binary atomically`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		prerelease, _ := cmd.Flags().GetBool("prerelease")
		return runUpdate(force, prerelease)
	},
}

type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func init() {
	updateCmd.Flags().BoolP("force", "f", false, "Force update even if already on latest version")
	updateCmd.Flags().Bool("prerelease", false, "Include pre-release versions")
}

func runUpdate(force, prerelease bool) error {
	fmt.Println("🔄 Checking for updates...")

	// Get current version
	currentVersion := strings.TrimPrefix(version, "v")
	fmt.Printf("Current version: %s\n", currentVersion)

	// Get latest release
	release, err := getLatestRelease(prerelease)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	fmt.Printf("Latest version: %s\n", latestVersion)

	// Compare versions
	if !force && !isNewerVersion(latestVersion, currentVersion) {
		fmt.Println("✅ You are already running the latest version!")
		return nil
	}

	fmt.Printf("📥 Updating to version %s...\n", latestVersion)

	// Download and install
	return downloadAndInstall(release)
}

func getLatestRelease(includePrerelease bool) (*GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases", githubAPIURL, repoOwner, updateRepoName)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}

	// Find the latest release (considering prerelease flag)
	for _, release := range releases {
		if !includePrerelease && release.Prerelease {
			continue
		}
		return &release, nil
	}

	return nil, fmt.Errorf("no suitable release found")
}

func isNewerVersion(latest, current string) bool {
	// Simple semantic version comparison (assumes vX.Y.Z format)
	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	// Pad with zeros if needed
	for len(latestParts) < 3 {
		latestParts = append(latestParts, "0")
	}
	for len(currentParts) < 3 {
		currentParts = append(currentParts, "0")
	}

	for i := 0; i < 3; i++ {
		latestNum, _ := strconv.Atoi(latestParts[i])
		currentNum, _ := strconv.Atoi(currentParts[i])

		if latestNum > currentNum {
			return true
		} else if latestNum < currentNum {
			return false
		}
	}

	return false // Equal versions
}

func downloadAndInstall(release *GitHubRelease) error {
	// Determine the asset name for current platform
	assetName := getAssetName()

	// Find the asset
	var downloadURL string
	var checksumURL string

	for _, asset := range release.Assets {
		switch asset.Name {
		case assetName:
			downloadURL = asset.BrowserDownloadURL
		case "checksums.txt":
			checksumURL = asset.BrowserDownloadURL
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no binary found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "amgctl-update-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Download binary
	fmt.Println("📦 Downloading binary...")
	binaryPath := filepath.Join(tempDir, assetName)
	if err := downloadFile(downloadURL, binaryPath); err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}

	// Download and verify checksum if available
	if checksumURL != "" {
		fmt.Println("🔍 Verifying checksum...")
		if err := verifyChecksum(binaryPath, checksumURL, assetName); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("✅ Checksum verified")
	}

	// Extract binary if it's compressed
	extractedPath := binaryPath
	if strings.HasSuffix(assetName, ".tar.gz") {
		fmt.Println("📂 Extracting binary...")
		extractedPath, err = extractBinary(binaryPath, tempDir)
		if err != nil {
			return fmt.Errorf("failed to extract binary: %w", err)
		}
	}

	// Make executable
	if err := os.Chmod(extractedPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Perform atomic replacement
	fmt.Println("🔄 Installing update...")
	if err := atomicReplace(extractedPath, currentExe); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	fmt.Printf("✅ Successfully updated to version %s!\n", release.TagName)
	fmt.Println("🚀 Restart your terminal or run the command again to use the new version.")

	return nil
}

func getAssetName() string {
	var suffix string
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}

	// Assume the release assets follow this naming pattern
	return fmt.Sprintf("amgctl-%s-%s%s", runtime.GOOS, runtime.GOARCH, suffix)
}

func downloadFile(url, filepath string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	_, err = io.Copy(out, resp.Body)
	return err
}

func verifyChecksum(binaryPath, checksumURL, assetName string) error {
	// Download checksums
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(checksumURL)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse checksums file (format: "hash filename")
	lines := strings.Split(string(checksumData), "\n")
	var expectedHash string

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == assetName {
			expectedHash = parts[0]
			break
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("checksum not found for %s", assetName)
	}

	// Calculate actual hash
	file, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

func extractBinary(tarPath, destDir string) (string, error) {
	file, err := os.Open(tarPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = gzr.Close()
	}()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Look for the binary file
		if header.Typeflag == tar.TypeReg && (strings.HasSuffix(header.Name, "amgctl") || strings.HasSuffix(header.Name, "amgctl.exe")) {
			extractedPath := filepath.Join(destDir, filepath.Base(header.Name))

			outFile, err := os.Create(extractedPath)
			if err != nil {
				return "", err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				return "", err
			}
			if err := outFile.Close(); err != nil {
				return "", err
			}

			return extractedPath, nil
		}
	}

	return "", fmt.Errorf("binary not found in archive")
}

func atomicReplace(newBinary, currentBinary string) error {
	// Create backup
	backupPath := currentBinary + ".backup"

	// On Windows, we need to handle the replacement differently
	if runtime.GOOS == "windows" {
		// Move current binary to backup
		if err := os.Rename(currentBinary, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}

		// Copy new binary to original location
		if err := copyFile(newBinary, currentBinary); err != nil {
			// Try to restore backup
			_ = os.Rename(backupPath, currentBinary)
			return fmt.Errorf("failed to install new binary: %w", err)
		}

		// Remove backup on success
		_ = os.Remove(backupPath)
	} else {
		// Unix-like systems: use atomic rename
		if err := copyFile(newBinary, currentBinary+".new"); err != nil {
			return fmt.Errorf("failed to copy new binary: %w", err)
		}

		if err := os.Rename(currentBinary+".new", currentBinary); err != nil {
			_ = os.Remove(currentBinary + ".new")
			return fmt.Errorf("failed to replace binary: %w", err)
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = destFile.Close()
	}()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
