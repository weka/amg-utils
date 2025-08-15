package cmd

import (
	"fmt"
	"os"
	"os/exec"
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
  amgctl docker launch --tensor-parallel-size 2 microsoft/DialoGPT-medium
  amgctl docker launch --docker-image "custom/vllm:v1.0" test-model
  amgctl docker launch --dry-run meta-llama/Llama-2-7b-chat-hf
  amgctl docker launch --no-enable-prefix-caching --lmcache-local-cpu my-model
  amgctl docker launch --max-num-batched-tokens 32768 --max-model-len 8192 my-model
  amgctl docker launch --hf-home "/custom/hf/cache" my-model
  amgctl docker launch --docker-arg "--memory=32g" --vllm-arg "--disable-log-stats" my-model
  amgctl docker launch --vllm-env "CUSTOM_VAR=value" --vllm-env "DEBUG=1" my-model`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		modelIdentifier := args[0]

		// Perform pre-flight checks
		if err := performPreflightChecks(); err != nil {
			return err
		}

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
		fmt.Printf("  Max Batched Tokens: %d\n", viper.GetInt("max-num-batched-tokens"))
		fmt.Printf("  Port: %d\n", viper.GetInt("port"))
		fmt.Printf("  LMCache Path: %s\n", viper.GetString("lmcache-path"))
		fmt.Printf("  LMCache Chunk Size: %d\n", viper.GetInt("lmcache-chunk-size"))
		fmt.Printf("  LMCache GDS Threads: %d\n", viper.GetInt("lmcache-gds-threads"))
		fmt.Printf("  LMCache cuFile Buffer Size: %s\n", viper.GetString("lmcache-cufile-buffer-size"))
		fmt.Printf("  LMCache Local CPU: %t\n", viper.GetBool("lmcache-local-cpu"))
		fmt.Printf("  LMCache Save Decode Cache: %t\n", viper.GetBool("lmcache-save-decode-cache"))
		fmt.Printf("  Hugging Face Cache: %s\n", viper.GetString("hf-home"))
		fmt.Printf("  vLLM Prefix Caching Disabled: %t\n", viper.GetBool("no-enable-prefix-caching"))

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

		// Detect InfiniBand device flags for Docker command generation
		ibFlags, err := hardware.GetInfinibandDeviceFlags()
		if err != nil {
			// Log warning but continue - InfiniBand is optional
			fmt.Printf("Warning: Failed to detect InfiniBand devices: %v\n", err)
			ibFlags = "" // Ensure empty string for Docker command generation
		}

		// Generate the Docker command
		dockerCmd, err := generateDockerCommand(
			modelIdentifier,
			cudaVisibleDevices,
			finalTensorParallelSize,
			ibFlags,
		)
		if err != nil {
			return fmt.Errorf("failed to generate Docker command: %v", err)
		}

		// Check if dry-run mode is enabled
		dryRun := viper.GetBool("dry-run")

		if dryRun {
			// Dry-run mode: display the command and exit
			fmt.Println("\n🔍 Dry Run Mode - Docker Command Preview:")
			fmt.Println("=====================================")
			fmt.Printf("%s\n", strings.Join(dockerCmd, " \\\n  "))
			fmt.Println("\n💡 To execute this command, run without --dry-run flag")
			return nil
		}

		// Normal mode: execute the command
		fmt.Println("\n🚀 Executing Docker Command...")
		return executeDockerCommand(dockerCmd)
	},
}

// generateDockerCommand assembles the complete docker run command as a slice of strings
func generateDockerCommand(modelIdentifier, cudaVisibleDevices string, tensorParallelSize int, ibFlags string) ([]string, error) {
	var cmd []string

	// Static parts: docker run with basic options
	cmd = append(cmd, "docker", "run", "-d")
	cmd = append(cmd, "--gpus", "all")
	cmd = append(cmd, "--runtime", "nvidia")
	cmd = append(cmd, "--network", "host")
	cmd = append(cmd, "--ipc", "host")

	// Add InfiniBand device flags if available
	if ibFlags != "" {
		// Split the flags string and add each --device flag
		deviceFlags := strings.Fields(ibFlags)
		cmd = append(cmd, deviceFlags...)
	}

	// Add volume mount for Weka filesystem
	wekaMount := viper.GetString("weka-mount")
	cmd = append(cmd, "-v", fmt.Sprintf("%s:/mnt/weka", wekaMount))

	// Add environment variables
	// CUDA_VISIBLE_DEVICES (if gpu-slots was used)
	if cudaVisibleDevices != "" && viper.GetString("gpu-slots") != "" {
		cmd = append(cmd, "-e", fmt.Sprintf("CUDA_VISIBLE_DEVICES=%s", cudaVisibleDevices))
	}

	// LMCache environment variables
	lmcachePath := viper.GetString("lmcache-path")
	lmcacheChunkSize := viper.GetInt("lmcache-chunk-size")
	lmcacheGdsThreads := viper.GetInt("lmcache-gds-threads")
	lmcacheCufileBufferSize := viper.GetString("lmcache-cufile-buffer-size")
	lmcacheLocalCpu := viper.GetBool("lmcache-local-cpu")
	lmcacheSaveDecodeCache := viper.GetBool("lmcache-save-decode-cache")

	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_PATH=%s", lmcachePath))
	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_CHUNK_SIZE=%d", lmcacheChunkSize))
	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_EXTRA_CONFIG={\"gds_io_threads\": %d}", lmcacheGdsThreads))
	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_CUFILE_BUFFER_SIZE=%s", lmcacheCufileBufferSize))
	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_LOCAL_CPU=%t", lmcacheLocalCpu))
	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_SAVE_DECODE_CACHE=%t", lmcacheSaveDecodeCache))

	// Hugging Face environment variables
	hfHome := viper.GetString("hf-home")
	cmd = append(cmd, "-e", fmt.Sprintf("HF_HOME=%s", hfHome))

	// Add custom environment variables from --vllm-env
	vllmEnvVars := viper.GetStringSlice("vllm-env")
	for _, envVar := range vllmEnvVars {
		if envVar != "" {
			cmd = append(cmd, "-e", envVar)
		}
	}

	// Add port mapping for vLLM API server
	port := viper.GetInt("port")
	cmd = append(cmd, "-p", fmt.Sprintf("%d:%d", port, port))

	// Add custom docker arguments from --docker-arg
	dockerArgs := viper.GetStringSlice("docker-arg")
	for _, arg := range dockerArgs {
		if arg != "" {
			cmd = append(cmd, arg)
		}
	}

	// Docker image name - use version-based default
	dockerImage := viper.GetString("docker-image")
	if dockerImage == "" {
		dockerImage = getDefaultDockerImage()
		// Auto-pull the image if it doesn't exist locally
		if err := pullImageIfNeeded(dockerImage); err != nil {
			return nil, fmt.Errorf("failed to ensure Docker image is available: %w", err)
		}
	}
	cmd = append(cmd, dockerImage)

	// vLLM serve command with all relevant flags
	vllmCmd := buildVllmCommand(modelIdentifier, tensorParallelSize)
	cmd = append(cmd, vllmCmd...)

	return cmd, nil
}

// buildVllmCommand constructs the vllm serve command with all relevant flags
func buildVllmCommand(modelIdentifier string, tensorParallelSize int) []string {
	var vllmCmd []string

	vllmCmd = append(vllmCmd, "vllm", "serve", modelIdentifier)

	// Add tensor parallel size
	vllmCmd = append(vllmCmd, "--tensor-parallel-size", strconv.Itoa(tensorParallelSize))

	// Add GPU memory utilization
	gpuMemUtil := viper.GetFloat64("gpu-mem-util")
	vllmCmd = append(vllmCmd, "--gpu-memory-utilization", fmt.Sprintf("%.2f", gpuMemUtil))

	// Add max sequences
	maxSequences := viper.GetInt("max-sequences")
	vllmCmd = append(vllmCmd, "--max-num-seqs", strconv.Itoa(maxSequences))

	// Add max model length
	maxModelLen := viper.GetInt("max-model-len")
	vllmCmd = append(vllmCmd, "--max-model-len", strconv.Itoa(maxModelLen))

	// Add max batched tokens
	maxBatchedTokens := viper.GetInt("max-num-batched-tokens")
	vllmCmd = append(vllmCmd, "--max-num-batched-tokens", strconv.Itoa(maxBatchedTokens))

	// Add port
	port := viper.GetInt("port")
	vllmCmd = append(vllmCmd, "--port", strconv.Itoa(port))

	// Add host binding
	vllmCmd = append(vllmCmd, "--host", "0.0.0.0")

	// Add prefix caching flag if disabled
	if viper.GetBool("no-enable-prefix-caching") {
		vllmCmd = append(vllmCmd, "--no-enable-prefix-caching")
	}

	// Add LMCache KV transfer configuration (always included)
	kvTransferConfig := `{"kv_connector":"LMCacheConnectorV1","kv_role":"kv_both","kv_connector_extra_config": {}}`
	vllmCmd = append(vllmCmd, "--kv-transfer-config", kvTransferConfig)

	// Add custom vLLM arguments from --vllm-arg
	vllmArgs := viper.GetStringSlice("vllm-arg")
	for _, arg := range vllmArgs {
		if arg != "" {
			vllmCmd = append(vllmCmd, arg)
		}
	}

	return vllmCmd
}

// executeDockerCommand executes the Docker command with real-time output streaming
func executeDockerCommand(dockerCmd []string) error {
	if len(dockerCmd) == 0 {
		return fmt.Errorf("docker command is empty")
	}

	// Create the command
	cmd := exec.Command(dockerCmd[0], dockerCmd[1:]...)

	// Stream stdout and stderr to the user's terminal in real-time
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up proper signal handling for the child process
	cmd.Stdin = os.Stdin

	// Display the command being executed (abbreviated version)
	fmt.Printf("Running: %s %s...\n", dockerCmd[0], strings.Join(dockerCmd[1:3], " "))

	// Execute the command
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("docker command failed: %w", err)
	}

	fmt.Println("\n✅ Docker container launched successfully!")
	return nil
}

func init() {
	dockerCmd.AddCommand(launchCmd)

	// Add persistent flags for launch configuration
	launchCmd.PersistentFlags().String("weka-mount", "/mnt/weka", "The Weka filesystem mount point on the host")
	launchCmd.PersistentFlags().Float64("gpu-mem-util", 0.8, "GPU memory utilization for vLLM")
	launchCmd.PersistentFlags().Int("max-sequences", 256, "The maximum number of sequences")
	launchCmd.PersistentFlags().Int("max-model-len", 16384, "The maximum model length")
	launchCmd.PersistentFlags().Int("max-num-batched-tokens", 16384, "The maximum number of batched tokens")
	launchCmd.PersistentFlags().Int("port", 8000, "The port for the vLLM API server")

	// Add GPU allocation flags
	launchCmd.PersistentFlags().String("gpu-slots", "", "Comma-separated list of GPU IDs to use (e.g., '0,1,2,3')")
	launchCmd.PersistentFlags().Int("tensor-parallel-size", 0, "Number of GPUs to use for tensor parallelism (used when --gpu-slots is not specified)")

	// Add Docker configuration flags
	launchCmd.PersistentFlags().String("docker-image", "", "Docker image to use for the vLLM container (defaults to sdimitro509/amg:<amgctl-version>, auto-pulled if needed)")
	launchCmd.PersistentFlags().Bool("dry-run", false, "Print the Docker command that would be executed without actually running it")

	// Add LMCache configuration flags
	launchCmd.PersistentFlags().String("lmcache-path", "/mnt/weka/cache", "Path for the cache within the Weka mount")
	launchCmd.PersistentFlags().Int("lmcache-chunk-size", 256, "LMCache chunk size")
	launchCmd.PersistentFlags().Int("lmcache-gds-threads", 32, "LMCache GDS threads")
	launchCmd.PersistentFlags().String("lmcache-cufile-buffer-size", "8192", "LMCache cuFile buffer size")
	launchCmd.PersistentFlags().Bool("lmcache-local-cpu", false, "Enable LMCache local CPU processing")
	launchCmd.PersistentFlags().Bool("lmcache-save-decode-cache", true, "Enable LMCache decode cache saving")

	// Add Hugging Face configuration flags
	launchCmd.PersistentFlags().String("hf-home", "/mnt/weka/hf_cache", "Hugging Face cache directory path")

	// Add vLLM configuration flags
	launchCmd.PersistentFlags().Bool("no-enable-prefix-caching", false, "Disable vLLM prefix caching")

	// Add escape hatch flags for advanced customization
	launchCmd.PersistentFlags().StringSlice("docker-arg", []string{}, "Additional arguments to pass to docker run command (repeatable)")
	launchCmd.PersistentFlags().StringSlice("vllm-arg", []string{}, "Additional arguments to pass to vllm serve command (repeatable)")
	launchCmd.PersistentFlags().StringSlice("vllm-env", []string{}, "Additional environment variables for vllm container in KEY=VALUE format (repeatable)")

	// Bind flags to Viper for configuration management
	// Note: viper.BindPFlag errors are typically only due to programming errors (nil flags)
	// and are safe to ignore in this context as flags are defined above
	_ = viper.BindPFlag("weka-mount", launchCmd.PersistentFlags().Lookup("weka-mount"))
	_ = viper.BindPFlag("gpu-mem-util", launchCmd.PersistentFlags().Lookup("gpu-mem-util"))
	_ = viper.BindPFlag("max-sequences", launchCmd.PersistentFlags().Lookup("max-sequences"))
	_ = viper.BindPFlag("max-model-len", launchCmd.PersistentFlags().Lookup("max-model-len"))
	_ = viper.BindPFlag("max-num-batched-tokens", launchCmd.PersistentFlags().Lookup("max-num-batched-tokens"))
	_ = viper.BindPFlag("port", launchCmd.PersistentFlags().Lookup("port"))
	_ = viper.BindPFlag("gpu-slots", launchCmd.PersistentFlags().Lookup("gpu-slots"))
	_ = viper.BindPFlag("tensor-parallel-size", launchCmd.PersistentFlags().Lookup("tensor-parallel-size"))
	_ = viper.BindPFlag("docker-image", launchCmd.PersistentFlags().Lookup("docker-image"))
	_ = viper.BindPFlag("dry-run", launchCmd.PersistentFlags().Lookup("dry-run"))
	_ = viper.BindPFlag("lmcache-path", launchCmd.PersistentFlags().Lookup("lmcache-path"))
	_ = viper.BindPFlag("lmcache-chunk-size", launchCmd.PersistentFlags().Lookup("lmcache-chunk-size"))
	_ = viper.BindPFlag("lmcache-gds-threads", launchCmd.PersistentFlags().Lookup("lmcache-gds-threads"))
	_ = viper.BindPFlag("lmcache-cufile-buffer-size", launchCmd.PersistentFlags().Lookup("lmcache-cufile-buffer-size"))
	_ = viper.BindPFlag("lmcache-local-cpu", launchCmd.PersistentFlags().Lookup("lmcache-local-cpu"))
	_ = viper.BindPFlag("lmcache-save-decode-cache", launchCmd.PersistentFlags().Lookup("lmcache-save-decode-cache"))
	_ = viper.BindPFlag("hf-home", launchCmd.PersistentFlags().Lookup("hf-home"))
	_ = viper.BindPFlag("no-enable-prefix-caching", launchCmd.PersistentFlags().Lookup("no-enable-prefix-caching"))
	_ = viper.BindPFlag("docker-arg", launchCmd.PersistentFlags().Lookup("docker-arg"))
	_ = viper.BindPFlag("vllm-arg", launchCmd.PersistentFlags().Lookup("vllm-arg"))
	_ = viper.BindPFlag("vllm-env", launchCmd.PersistentFlags().Lookup("vllm-env"))
}

// performPreflightChecks validates system requirements and configuration before execution
func performPreflightChecks() error {
	// Check if docker command exists in PATH
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker command not found in PATH. Please install Docker and ensure it's available in your system PATH")
	}

	// Check if weka-mount path exists
	wekaMount := viper.GetString("weka-mount")
	if wekaMount != "" {
		if _, err := os.Stat(wekaMount); os.IsNotExist(err) {
			return fmt.Errorf("weka mount path '%s' does not exist. Please ensure the path exists or specify a different --weka-mount", wekaMount)
		} else if err != nil {
			return fmt.Errorf("failed to access weka mount path '%s': %v", wekaMount, err)
		}
	}

	// Check if hf-home directory exists
	hfHome := viper.GetString("hf-home")
	if hfHome != "" {
		if _, err := os.Stat(hfHome); os.IsNotExist(err) {
			return fmt.Errorf("hugging Face cache directory '%s' does not exist. Please create the directory or specify a different --hf-home", hfHome)
		} else if err != nil {
			return fmt.Errorf("failed to access Hugging Face cache directory '%s': %v", hfHome, err)
		}
	}

	return nil
}
