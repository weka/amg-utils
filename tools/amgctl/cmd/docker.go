package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Docker management commands",
	Long:  `Manage Docker containers, images, and configurations for AMG.`,
}

var dockerPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull Docker image",
	Long:  `Pull the AMG Docker image from the registry.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		version := viper.GetString("version")
		name := viper.GetString("name")
		return runDockerPull(version, name)
	},
}

func init() {
	dockerCmd.AddCommand(dockerPullCmd)
	dockerPullCmd.Flags().StringP("version", "v", version, "Version of the AMG image to pull")
	dockerPullCmd.Flags().StringP("name", "n", "", "Local name to tag the pulled image (optional)")
	cobra.CheckErr(viper.BindPFlag("version", dockerPullCmd.Flags().Lookup("version")))
	cobra.CheckErr(viper.BindPFlag("name", dockerPullCmd.Flags().Lookup("name")))
}

func runDockerPull(version, name string) error {
	fmt.Println("🐳 Docker Pull Command")

	// Construct the image name
	imageName := fmt.Sprintf("sdimitro509/amg:v%s", version)

	// Use the fallback logic to pull the image
	actualImageName, err := pullImageWithFallback(imageName)
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// If a custom name is provided, tag the image
	if name != "" {
		fmt.Printf("Tagging image as: %s\n", name)
		tagCmd := exec.Command("docker", "tag", actualImageName, name)
		tagCmd.Stdout = os.Stdout
		tagCmd.Stderr = os.Stderr

		err = tagCmd.Run()
		if err != nil {
			return fmt.Errorf("failed to tag image as %s: %w", name, err)
		}

		fmt.Printf("✅ Successfully tagged image as: %s\n", name)
	}

	return nil
}

// getDefaultDockerImage returns the default Docker image based on amgctl version
func getDefaultDockerImage() string {
	return fmt.Sprintf("sdimitro509/amg:v%s", version)
}

// imageExists checks if a Docker image exists locally
func imageExists(imageName string) bool {
	cmd := exec.Command("docker", "image", "inspect", imageName)
	return cmd.Run() == nil
}

// pullImageWithFallback attempts to pull a Docker image with fallback to latest
func pullImageWithFallback(imageName string) (string, error) {
	fmt.Printf("Attempting to pull image: %s\n", imageName)

	cmd := exec.Command("docker", "pull", imageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		// Extract base image name without tag for fallback
		baseImage := "sdimitro509/amg"
		latestImage := baseImage + ":latest"

		fmt.Printf("❌ Failed to pull %s\n", imageName)
		fmt.Printf("🔄 Falling back to latest version: %s\n", latestImage)

		// Try pulling the latest tag
		latestCmd := exec.Command("docker", "pull", latestImage)
		latestCmd.Stdout = os.Stdout
		latestCmd.Stderr = os.Stderr

		err = latestCmd.Run()
		if err != nil {
			return "", fmt.Errorf("failed to pull both %s and %s: %w", imageName, latestImage, err)
		}

		fmt.Printf("✅ Successfully pulled fallback image: %s\n", latestImage)
		return latestImage, nil
	}

	fmt.Printf("✅ Successfully pulled image: %s\n", imageName)
	return imageName, nil
}


