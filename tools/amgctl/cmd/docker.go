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
	dockerPullCmd.Flags().StringP("version", "v", "0.1.0", "Version of the AMG image to pull")
	dockerPullCmd.Flags().StringP("name", "n", "", "Local name to tag the pulled image (optional)")
	cobra.CheckErr(viper.BindPFlag("version", dockerPullCmd.Flags().Lookup("version")))
	cobra.CheckErr(viper.BindPFlag("name", dockerPullCmd.Flags().Lookup("name")))
}

func runDockerPull(version, name string) error {
	fmt.Println("🐳 Docker Pull Command")

	// Construct the image name
	imageName := fmt.Sprintf("sdimitro509/amg:%s", version)
	fmt.Printf("Pulling image: %s\n", imageName)

	// Execute docker pull command
	cmd := exec.Command("docker", "pull", imageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}

	fmt.Printf("✅ Successfully pulled image: %s\n", imageName)

	// If a custom name is provided, tag the image
	if name != "" {
		fmt.Printf("Tagging image as: %s\n", name)
		tagCmd := exec.Command("docker", "tag", imageName, name)
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
