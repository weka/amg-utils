package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/weka/amg-utils/tools/amgctl/internal/hardware"
)

var launchCmd = &cobra.Command{
	Use:   "launch <model_identifier>",
	Short: "Launch an AMG container with the specified model",
	Long: `Launch an AMG container with specified configurations for the given model.

The model_identifier is a required argument that specifies which model to deploy.

Examples:
  amgctl docker launch meta-llama/Llama-2-7b-chat-hf
  amgctl docker launch microsoft/DialoGPT-medium
  amgctl docker launch --gpu-mem-util 0.8 --port 8080 openai-gpt-3.5-turbo
  amgctl docker launch --gpu-slots "0,1,2,3" meta-llama/Llama-2-7b-chat-hf
  amgctl docker launch --tensor-parallel-size 2 microsoft/DialoGPT-medium`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		modelIdentifier := args[0]
		fmt.Printf("amgctl docker launch called with model: %s\n", modelIdentifier)

		// Handle GPU allocation logic
		gpuSlots := viper.GetString("gpu-slots")
		tensorParallelSize := viper.GetInt("tensor-parallel-size")
		var cudaVisibleDevices string
		var finalTensorParallelSize int

		if gpuSlots != "" {
			// Parse comma-separated GPU slots
			gpuIDs := strings.Split(gpuSlots, ",")
			var validGpuIDs []string

			// Validate and clean up GPU IDs
			for _, id := range gpuIDs {
				id = strings.TrimSpace(id)
				if _, err := strconv.Atoi(id); err != nil {
					return fmt.Errorf("invalid GPU ID '%s' in --gpu-slots: must be numeric", id)
				}
				validGpuIDs = append(validGpuIDs, id)
			}

			if len(validGpuIDs) == 0 {
				return fmt.Errorf("--gpu-slots cannot be empty")
			}

			cudaVisibleDevices = strings.Join(validGpuIDs, ",")
			finalTensorParallelSize = len(validGpuIDs)
		} else {
			// Use tensor-parallel-size flag or auto-detect
			if tensorParallelSize > 0 {
				finalTensorParallelSize = tensorParallelSize
			} else {
				// Auto-detect GPU count
				gpuCount, err := hardware.GetGpuCount()
				if err != nil {
					return fmt.Errorf("failed to auto-detect GPU count: %v", err)
				}
				finalTensorParallelSize = gpuCount
			}

			// Set CUDA_VISIBLE_DEVICES to use all available GPUs up to tensor-parallel-size
			var deviceIDs []string
			for i := 0; i < finalTensorParallelSize; i++ {
				deviceIDs = append(deviceIDs, strconv.Itoa(i))
			}
			cudaVisibleDevices = strings.Join(deviceIDs, ",")
		}

		// Display configuration
		fmt.Println("\nLaunch Configuration:")
		fmt.Printf("  Model: %s\n", modelIdentifier)
		fmt.Printf("  Weka Mount: %s\n", viper.GetString("weka-mount"))
		fmt.Printf("  GPU Memory Utilization: %.2f\n", viper.GetFloat64("gpu-mem-util"))
		fmt.Printf("  Max Sequences: %d\n", viper.GetInt("max-sequences"))
		fmt.Printf("  Max Model Length: %d\n", viper.GetInt("max-model-len"))
		fmt.Printf("  Port: %d\n", viper.GetInt("port"))
		fmt.Printf("  LMCache Path: %s\n", viper.GetString("lmcache-path"))
		fmt.Printf("  LMCache Chunk Size: %d\n", viper.GetInt("lmcache-chunk-size"))
		fmt.Printf("  LMCache GDS Threads: %d\n", viper.GetInt("lmcache-gds-threads"))

		// Display GPU allocation settings
		fmt.Println("\nGPU Allocation:")
		if gpuSlots != "" {
			fmt.Printf("  GPU Slots (manual): %s\n", gpuSlots)
		} else if tensorParallelSize > 0 {
			fmt.Printf("  Tensor Parallel Size (manual): %d\n", tensorParallelSize)
		} else {
			fmt.Printf("  Tensor Parallel Size (auto-detected): %d\n", finalTensorParallelSize)
		}
		fmt.Printf("  CUDA_VISIBLE_DEVICES: %s\n", cudaVisibleDevices)
		fmt.Printf("  Final Tensor Parallel Size: %d\n", finalTensorParallelSize)

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

	// Add persistent flags for launch configuration
	launchCmd.PersistentFlags().String("weka-mount", "/mnt/weka", "The Weka filesystem mount point on the host")
	launchCmd.PersistentFlags().Float64("gpu-mem-util", 0.9, "GPU memory utilization for vLLM")
	launchCmd.PersistentFlags().Int("max-sequences", 256, "The maximum number of sequences")
	launchCmd.PersistentFlags().Int("max-model-len", 16384, "The maximum model length")
	launchCmd.PersistentFlags().Int("port", 8000, "The port for the vLLM API server")

	// Add GPU allocation flags
	launchCmd.PersistentFlags().String("gpu-slots", "", "Comma-separated list of GPU IDs to use (e.g., '0,1,2,3')")
	launchCmd.PersistentFlags().Int("tensor-parallel-size", 0, "Number of GPUs to use for tensor parallelism (used when --gpu-slots is not specified)")

	// Add LMCache configuration flags
	launchCmd.PersistentFlags().String("lmcache-path", "/mnt/weka/cache", "Path for the cache within the Weka mount")
	launchCmd.PersistentFlags().Int("lmcache-chunk-size", 256, "LMCache chunk size")
	launchCmd.PersistentFlags().Int("lmcache-gds-threads", 32, "LMCache GDS threads")

	// Bind flags to Viper for configuration management
	// Note: viper.BindPFlag errors are typically only due to programming errors (nil flags)
	// and are safe to ignore in this context as flags are defined above
	_ = viper.BindPFlag("weka-mount", launchCmd.PersistentFlags().Lookup("weka-mount"))
	_ = viper.BindPFlag("gpu-mem-util", launchCmd.PersistentFlags().Lookup("gpu-mem-util"))
	_ = viper.BindPFlag("max-sequences", launchCmd.PersistentFlags().Lookup("max-sequences"))
	_ = viper.BindPFlag("max-model-len", launchCmd.PersistentFlags().Lookup("max-model-len"))
	_ = viper.BindPFlag("port", launchCmd.PersistentFlags().Lookup("port"))
	_ = viper.BindPFlag("gpu-slots", launchCmd.PersistentFlags().Lookup("gpu-slots"))
	_ = viper.BindPFlag("tensor-parallel-size", launchCmd.PersistentFlags().Lookup("tensor-parallel-size"))
	_ = viper.BindPFlag("lmcache-path", launchCmd.PersistentFlags().Lookup("lmcache-path"))
	_ = viper.BindPFlag("lmcache-chunk-size", launchCmd.PersistentFlags().Lookup("lmcache-chunk-size"))
	_ = viper.BindPFlag("lmcache-gds-threads", launchCmd.PersistentFlags().Lookup("lmcache-gds-threads"))
}
