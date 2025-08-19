package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update amgctl to the latest version",
	Long:  `Check for updates and install the latest version of amgctl.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		return Update(force)
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
	updateCmd.Flags().BoolP("force", "f", false, "Force update even if the version is the same")
}

// Update check for a new version on GitHub and self-update
func Update(force bool) error {
	fmt.Println("🔄 Checking for updates...")

	latestVersion, releaseURL, err := getLatestVersion(false)
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	fmt.Printf("Latest version: %s\n", latestVersion)
	fmt.Printf("Current version: %s\n", version)

	if !force && latestVersion == version {
		fmt.Println("✅ You are already running the latest version!")
		return nil
	}

	fmt.Println("⬇️  Downloading new version...")
	return downloadAndInstall(releaseURL)
}

func getLatestVersion(prerelease bool) (string, string, error) {
	var url string
	if prerelease {
		url = "https://api.github.com/repos/weka/amg-utils/releases"
	} else {
		url = "https://api.github.com/repos/weka/amg-utils/releases/latest"
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			panic(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to fetch releases: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			BrowserDownloadURL string `json:"browser_download_url"`
			Name               string `json:"name"`
		} `json:"assets"`
	}

	if prerelease {
		var releases []struct {
			TagName    string `json:"tag_name"`
			Prerelease bool   `json:"prerelease"`
			Assets     []struct {
				BrowserDownloadURL string `json:"browser_download_url"`
				Name               string `json:"name"`
			} `json:"assets"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return "", "", fmt.Errorf("failed to parse releases: %w", err)
		}
		if len(releases) > 0 {
			release.TagName = releases[0].TagName
			release.Assets = releases[0].Assets
		}
	} else {
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return "", "", fmt.Errorf("failed to parse release: %w", err)
		}
	}

	assetName := fmt.Sprintf("amgctl-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			return release.TagName, asset.BrowserDownloadURL, nil
		}
	}

	return "", "", fmt.Errorf("no suitable asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
}

func downloadAndInstall(url string) error {
	// Get the current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not get executable path: %w", err)
	}

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "amgctl-update-")
	if err != nil {
		return fmt.Errorf("could not create temporary file: %w", err)
	}
	defer func() {
		// Only close if not already closed
		if tmpFile != nil {
			_ = tmpFile.Close() // Ignore error in defer
		}
	}()

	// Download the new executable
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("could not download new version: %w", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			panic(err)
		}
	}()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("could not write to temporary file: %w", err)
	}

	// Make the temporary file executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("could not make temporary file executable: %w", err)
	}

	// Store the temp file name before closing
	tmpFileName := tmpFile.Name()

	// Close the file so it can be renamed
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("could not close temporary file: %w", err)
	}
	tmpFile = nil // Mark as closed for defer

	// Replace the old executable with the new one
	if err := os.Rename(tmpFileName, exePath); err != nil {
		return fmt.Errorf("could not replace executable: %w", err)
	}

	fmt.Println("✅ amgctl updated successfully!")
	return nil
}
