package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weka/amg-utils/tools/amgctl/internal/hardware"
)

var hwCmd = &cobra.Command{
	Use:   "hw",
	Short: "Hardware information and management commands",
	Long:  `Display and manage hardware information for AMG deployments.`,
}

var hwShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show hardware information",
	Long: `Display detailed information about available hardware including GPUs and InfiniBand devices.

This command provides comprehensive hardware discovery for AMG deployments,
showing GPU details and InfiniBand device information that will be used
for container deployments.

Examples:
  amgctl hw show`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🖥️  Hardware Information")
		fmt.Println("========================")

		// Display GPU information
		if err := displayGpuInfo(); err != nil {
			fmt.Printf("Warning: Failed to detect NVIDIA GPUs: %v\n", err)
			fmt.Println("This may be expected if NVIDIA drivers are not installed or no GPUs are present.")
		}

		fmt.Println() // Add spacing between sections

		// Display InfiniBand information
		if err := displayInfinibandInfo(); err != nil {
			fmt.Printf("Warning: Failed to detect InfiniBand devices: %v\n", err)
			fmt.Println("This may be expected if InfiniBand devices are not present or drivers are not installed.")
		}

		return nil
	},
}

func init() {
	hwCmd.AddCommand(hwShowCmd)
}

// displayGpuInfo shows GPU detection and details
func displayGpuInfo() error {
	gpuCount, err := hardware.GetGpuCount()
	if err != nil {
		return err
	}

	fmt.Printf("Detected %d NVIDIA GPU(s)\n", gpuCount)

	// Display detailed GPU information if GPUs are available
	if gpuCount > 0 {
		gpuInfo, infoErr := hardware.GetGpuInfo()
		if infoErr != nil {
			return fmt.Errorf("failed to get GPU details: %w", infoErr)
		}

		fmt.Println("GPU Details:")
		for _, info := range gpuInfo {
			fmt.Printf("  %s\n", info)
		}
	}

	return nil
}

// displayInfinibandInfo shows InfiniBand device detection and details
func displayInfinibandInfo() error {
	ibFlags, err := hardware.GetInfinibandDeviceFlags()
	if err != nil {
		return err
	}

	if ibFlags == "" {
		fmt.Println("No InfiniBand devices detected")
		return nil
	}

	// Display detailed InfiniBand information
	ibInfo, infoErr := hardware.GetInfinibandDeviceInfo()
	if infoErr != nil {
		return fmt.Errorf("failed to get InfiniBand device details: %w", infoErr)
	}

	fmt.Println("InfiniBand Device Details:")
	for _, info := range ibInfo {
		fmt.Printf("  %s\n", info)
	}

	return nil
}
