package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weka/amg-utils/tools/amgctl/internal/hardware"
)

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch an AMG container",
	Long:  `Launch an AMG container with specified configurations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("amgctl docker launch called")

		// Detect and display NVIDIA GPU count
		gpuCount, err := hardware.GetGpuCount()
		if err != nil {
			fmt.Printf("Warning: Failed to detect NVIDIA GPUs: %v\n", err)
			fmt.Println("This may be expected if NVIDIA drivers are not installed or no GPUs are present.")
		} else {
			fmt.Printf("Detected %d NVIDIA GPU(s)\n", gpuCount)

			// Optionally display detailed GPU information
			if gpuCount > 0 {
				gpuInfo, infoErr := hardware.GetGpuInfo()
				if infoErr != nil {
					fmt.Printf("Warning: Failed to get GPU details: %v\n", infoErr)
				} else {
					fmt.Println("GPU Details:")
					for _, info := range gpuInfo {
						fmt.Printf("  %s\n", info)
					}
				}
			}
		}

		// Detect and display InfiniBand device flags
		ibFlags, err := hardware.GetInfinibandDeviceFlags()
		if err != nil {
			fmt.Printf("Warning: Failed to detect InfiniBand devices: %v\n", err)
			fmt.Println("This may be expected if InfiniBand devices are not present or drivers are not installed.")
		} else if ibFlags != "" {
			fmt.Println("\nInfiniBand Docker Device Flags:")
			fmt.Printf("%s\n", ibFlags)

			// Optionally display detailed InfiniBand information
			ibInfo, infoErr := hardware.GetInfinibandDeviceInfo()
			if infoErr != nil {
				fmt.Printf("Warning: Failed to get InfiniBand device details: %v\n", infoErr)
			} else {
				fmt.Println("\nInfiniBand Device Details:")
				for _, info := range ibInfo {
					fmt.Printf("  %s\n", info)
				}
			}
		} else {
			fmt.Println("No InfiniBand devices detected")
		}

		return nil
	},
}

func init() {
	dockerCmd.AddCommand(launchCmd)
}
