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

var dockerLaunchCmd = &cobra.Command{
	Use:   "launch <model_identifier>",
	Short: "Launch an AMG container with the specified model",
	Long: `Launch an AMG container with specified configurations for the given model.
This command runs 'amgctl host launch' inside a Docker container.

The model_identifier is a required argument that specifies which model to deploy.

By default, cufile.json is automatically created and configured with optimal settings
inside the container. Use --skip-cufile-configure to disable this behavior.

Examples:
  amgctl docker launch meta-llama/Llama-2-7b-chat-hf
  amgctl docker launch microsoft/DialoGPT-medium
  amgctl docker launch --gpu-mem-util 0.8 --port 8080 openai-gpt-3.5-turbo
  amgctl docker launch --gpu-slots "0,1,2,3" meta-llama/Llama-2-7b-chat-hf
  amgctl docker launch --tensor-parallel-size 2 microsoft/DialoGPT-medium
  amgctl docker launch --docker-image "custom/vllm:v1.0" test-model
  amgctl docker launch --dry-run meta-llama/Llama-2-7b-chat-hf
  amgctl docker launch --generate-cufile-json meta-llama/Llama-2-7b-chat-hf
  amgctl docker launch --no-enable-prefix-caching --lmcache-local-cpu my-model
  amgctl docker launch --max-num-batched-tokens 32768 --max-model-len 8192 my-model
  amgctl docker launch --hf-home "/custom/hf/cache" my-model
  amgctl docker launch --prometheus-multiproc-dir "/tmp/prometheus" my-model
  amgctl docker launch --docker-arg "--memory=32g" --vllm-arg "--disable-log-stats" my-model
  amgctl docker launch --vllm-env "CUSTOM_VAR=value" --vllm-env "DEBUG=1" my-model
  amgctl docker launch --skip-safefasttensors my-model`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		modelIdentifier := args[0]

		// Perform pre-flight checks
		if err := performDockerPreflightChecks(); err != nil {
			return err
		}

		// Handle GPU allocation logic
		gpuSlots := viper.GetString("gpu-slots")
		tensorParallelSize := viper.GetInt("tensor-parallel-size")
		var cudaVisibleDevices string
		var finalTensorParallelSize int

		if gpuSlots != "" {
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
		fmt.Printf("  LMCache Path: %s\n", viper.GetString("lmcache-weka-path"))
		fmt.Printf("  LMCache Chunk Size: %d\n", viper.GetInt("lmcache-chunk-size"))
		fmt.Printf("  LMCache GDS Threads: %d\n", viper.GetInt("lmcache-gds-threads"))
		fmt.Printf("  LMCache cuFile Buffer Size: %s\n", viper.GetString("lmcache-cufile-buffer-size"))
		fmt.Printf("  LMCache Local CPU: %t\n", viper.GetBool("lmcache-local-cpu"))
		fmt.Printf("  LMCache Save Decode Cache: %t\n", viper.GetBool("lmcache-save-decode-cache"))
		fmt.Printf("  Hugging Face Cache: %s\n", viper.GetString("hf-home"))
		fmt.Printf("  Prometheus Multiproc Dir: %s\n", viper.GetString("prometheus-multiproc-dir"))
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
		dockerCmd, err := generateDockerHostLaunchCommand(
			cmd,
			modelIdentifier,
			cudaVisibleDevices,
			finalTensorParallelSize,
			ibFlags,
		)
		if err != nil {
			return fmt.Errorf("failed to generate Docker command: %v", err)
		}

		// Check if dry-run mode is enabled
		dryRun, _ := cmd.Flags().GetBool("dry-run")

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
		return executeDockerLaunchCommand(dockerCmd)
	},
}

// generateHostLaunchCommand assembles the complete amgctl host launch command as a slice of strings
func generateDockerHostLaunchCommand(cobraCmd *cobra.Command, modelIdentifier, cudaVisibleDevices string, tensorParallelSize int, ibFlags string) ([]string, error) {
	var cmd []string

	// Static parts: docker run with basic options
	cmd = append(cmd, "docker", "run", "-d")
	cmd = append(cmd, "--gpus", "all")
	cmd = append(cmd, "--runtime", "nvidia")
	cmd = append(cmd, "--network", "host")
	cmd = append(cmd, "--ipc", "host")

	if ibFlags != "" {
		// Split the flags string and add each --device flag
		deviceFlags := strings.Fields(ibFlags)
		cmd = append(cmd, deviceFlags...)
	}

	wekaMount := viper.GetString("weka-mount")
	cmd = append(cmd, "-v", fmt.Sprintf("%s:/mnt/weka", wekaMount))

	// CUDA_VISIBLE_DEVICES (if gpu-slots was used)
	if cudaVisibleDevices != "" && viper.GetString("gpu-slots") != "" {
		cmd = append(cmd, "-e", fmt.Sprintf("CUDA_VISIBLE_DEVICES=%s", cudaVisibleDevices))
	}

	// LMCache environment variables
	lmcacheWekaPath := viper.GetString("lmcache-weka-path")
	lmcacheChunkSize := viper.GetInt("lmcache-chunk-size")
	lmcacheGdsThreads := viper.GetInt("lmcache-gds-threads")
	lmcacheCufileBufferSize := viper.GetString("lmcache-cufile-buffer-size")
	lmcacheLocalCpu := viper.GetBool("lmcache-local-cpu")
	lmcacheSaveDecodeCache := viper.GetBool("lmcache-save-decode-cache")

	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_WEKA_PATH=%s", lmcacheWekaPath))
	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_CHUNK_SIZE=%d", lmcacheChunkSize))
	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_EXTRA_CONFIG={\"gds_io_threads\": %d}", lmcacheGdsThreads))
	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_CUFILE_BUFFER_SIZE=%s", lmcacheCufileBufferSize))
	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_LOCAL_CPU=%t", lmcacheLocalCpu))
	cmd = append(cmd, "-e", fmt.Sprintf("LMCACHE_SAVE_DECODE_CACHE=%t", lmcacheSaveDecodeCache))

	// Hugging Face environment variables
	hfHome := viper.GetString("hf-home")
	cmd = append(cmd, "-e", fmt.Sprintf("HF_HOME=%s", hfHome))

	// Prometheus environment variables
	prometheusDir := viper.GetString("prometheus-multiproc-dir")
	cmd = append(cmd, "-e", fmt.Sprintf("PROMETHEUS_MULTIPROC_DIR=%s", prometheusDir))

	// Add USE_FASTSAFETENSOR environment variable unless --skip-safefasttensors is set
	if !viper.GetBool("skip-safefasttensors") {
		cmd = append(cmd, "-e", "USE_FASTSAFETENSOR=true")
	}

	// Add custom environment variables from --vllm-env
	vllmEnvVars := viper.GetStringSlice("vllm-env")
	for _, envVar := range vllmEnvVars {
		if envVar != "" {
			cmd = append(cmd, "-e", envVar)
		}
	}

	port := viper.GetInt("port")
	cmd = append(cmd, "-p", fmt.Sprintf("%d:%d", port, port))

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
		// Auto-pull the image if it doesn't exist locally, handling fallback
		if !imageExists(dockerImage) {
			actualImage, err := pullImageWithFallback(dockerImage)
			if err != nil {
				return nil, fmt.Errorf("failed to ensure Docker image is available: %w", err)
			}
			// Use the actual image that was pulled (could be latest if fallback occurred)
			dockerImage = actualImage
		}
	}
	cmd = append(cmd, dockerImage)

	// amgctl host launch command with all relevant flags
	hostLaunchCmd := buildDockerHostLaunchCommand(cobraCmd, modelIdentifier, tensorParallelSize)
	cmd = append(cmd, hostLaunchCmd...)

	return cmd, nil
}

// buildHostLaunchCommand constructs the amgctl host launch command with all relevant flags
func buildDockerHostLaunchCommand(cobraCmd *cobra.Command, modelIdentifier string, tensorParallelSize int) []string {
	var hostCmd []string

	// Use amgctl host launch command
	hostCmd = append(hostCmd, "amgctl", "host", "launch", modelIdentifier)

	// Add GPU allocation flags
	if cobraCmd.Flags().Changed("gpu-slots") {
		gpuSlots := viper.GetString("gpu-slots")
		hostCmd = append(hostCmd, "--gpu-slots", gpuSlots)
	} else if tensorParallelSize > 0 {
		hostCmd = append(hostCmd, "--tensor-parallel-size", strconv.Itoa(tensorParallelSize))
	}

	// Add GPU memory utilization
	if cobraCmd.Flags().Changed("gpu-mem-util") {
		gpuMemUtil := viper.GetFloat64("gpu-mem-util")
		hostCmd = append(hostCmd, "--gpu-mem-util", fmt.Sprintf("%.2f", gpuMemUtil))
	}

	// Add vLLM configuration flags
	if cobraCmd.Flags().Changed("max-sequences") {
		maxSequences := viper.GetInt("max-sequences")
		hostCmd = append(hostCmd, "--max-sequences", strconv.Itoa(maxSequences))
	}

	if cobraCmd.Flags().Changed("max-model-len") {
		maxModelLen := viper.GetInt("max-model-len")
		hostCmd = append(hostCmd, "--max-model-len", strconv.Itoa(maxModelLen))
	}

	if cobraCmd.Flags().Changed("max-num-batched-tokens") {
		maxBatchedTokens := viper.GetInt("max-num-batched-tokens")
		hostCmd = append(hostCmd, "--max-num-batched-tokens", strconv.Itoa(maxBatchedTokens))
	}

	if cobraCmd.Flags().Changed("port") {
		port := viper.GetInt("port")
		hostCmd = append(hostCmd, "--port", strconv.Itoa(port))
	}

	// Add Weka mount if specified
	if cobraCmd.Flags().Changed("weka-mount") {
		wekaMount := viper.GetString("weka-mount")
		hostCmd = append(hostCmd, "--weka-mount", wekaMount)
	}

	// Add LMCache configuration flags
	if cobraCmd.Flags().Changed("lmcache-weka-path") {
		lmcacheWekaPath := viper.GetString("lmcache-weka-path")
		hostCmd = append(hostCmd, "--lmcache-weka-path", lmcacheWekaPath)
	}

	if cobraCmd.Flags().Changed("lmcache-chunk-size") {
		lmcacheChunkSize := viper.GetInt("lmcache-chunk-size")
		hostCmd = append(hostCmd, "--lmcache-chunk-size", strconv.Itoa(lmcacheChunkSize))
	}

	if cobraCmd.Flags().Changed("lmcache-gds-threads") {
		lmcacheGdsThreads := viper.GetInt("lmcache-gds-threads")
		hostCmd = append(hostCmd, "--lmcache-gds-threads", strconv.Itoa(lmcacheGdsThreads))
	}

	if cobraCmd.Flags().Changed("lmcache-cufile-buffer-size") {
		lmcacheCufileBufferSize := viper.GetString("lmcache-cufile-buffer-size")
		hostCmd = append(hostCmd, "--lmcache-cufile-buffer-size", lmcacheCufileBufferSize)
	}

	if cobraCmd.Flags().Changed("lmcache-local-cpu") {
		hostCmd = append(hostCmd, "--lmcache-local-cpu")
	}

	if cobraCmd.Flags().Changed("lmcache-save-decode-cache") {
		// Pass the value explicitly since the host launch default is true, but docker launch default is now false
		hostCmd = append(hostCmd, "--lmcache-save-decode-cache", strconv.FormatBool(viper.GetBool("lmcache-save-decode-cache")))
	}

	// Add Hugging Face configuration
	if cobraCmd.Flags().Changed("hf-home") {
		hfHome := viper.GetString("hf-home")
		hostCmd = append(hostCmd, "--hf-home", hfHome)
	}

	// Add Prometheus configuration
	if cobraCmd.Flags().Changed("prometheus-multiproc-dir") {
		prometheusDir := viper.GetString("prometheus-multiproc-dir")
		hostCmd = append(hostCmd, "--prometheus-multiproc-dir", prometheusDir)
	}

	// Add prefix caching flag if disabled
	if cobraCmd.Flags().Changed("no-enable-prefix-caching") {
		hostCmd = append(hostCmd, "--no-enable-prefix-caching")
	}

	// Add skip fastsafetensors flag if set
	if cobraCmd.Flags().Changed("skip-safefasttensors") {
		hostCmd = append(hostCmd, "--skip-safefasttensors")
	}

	// Add skip cufile configure flag if set
	if cobraCmd.Flags().Changed("skip-cufile-configure") {
		hostCmd = append(hostCmd, "--skip-cufile-configure")
	}

	// Add custom vLLM arguments from --vllm-arg
	if cobraCmd.Flags().Changed("vllm-arg") {
		vllmArgs := viper.GetStringSlice("vllm-arg")
		for _, arg := range vllmArgs {
			if arg != "" {
				hostCmd = append(hostCmd, "--vllm-arg", arg)
			}
		}
	}

	// Add custom environment variables from --vllm-env
	if cobraCmd.Flags().Changed("vllm-env") {
		vllmEnvVars := viper.GetStringSlice("vllm-env")
		for _, envVar := range vllmEnvVars {
			if envVar != "" {
				hostCmd = append(hostCmd, "--vllm-env", envVar)
			}
		}
	}

	// Note: dry-run is handled by the docker launch command itself, not propagated to host launch

	return hostCmd
}

// executeDockerCommand executes the Docker command with real-time output streaming
func executeDockerLaunchCommand(dockerCmd []string) error {
	if len(dockerCmd) == 0 {
		return fmt.Errorf("docker command is empty")
	}

	// Create the command
	cmd := exec.Command(dockerCmd[0], dockerCmd[1:]...)

	// Capture the container ID from stdout
	var stdout strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	// Set up proper signal handling for the child process
	cmd.Stdin = os.Stdin

	// Display the command being executed (abbreviated version)
	fmt.Printf("Running: %s %s...\n", dockerCmd[0], strings.Join(dockerCmd[1:3], " "))

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("docker command failed: %w", err)
	}

	// Get the container ID from stdout
	containerID := strings.TrimSpace(stdout.String())
	if containerID == "" {
		fmt.Println("\n✅ Docker container launched successfully!")
		fmt.Println("💡 Note: To check container status, use: docker ps")
		fmt.Println("💡 To view container logs, use: docker logs <container_id>")
		return nil
	}

	fmt.Printf("\n✅ Docker container launched successfully!\n")
	fmt.Printf("📦 Container ID: %s\n", containerID)
	fmt.Printf("💡 To check container status: docker ps\n")
	fmt.Printf("💡 To view container logs: docker logs %s\n", containerID)
	fmt.Printf("💡 To stop the container: docker stop %s\n", containerID)

	return nil
}

func init() {
	dockerCmd.AddCommand(dockerLaunchCmd)

	// Add persistent flags for launch configuration
	// Note: No default values - these are passed through to 'amgctl host launch' which defines the defaults
	dockerLaunchCmd.PersistentFlags().String("weka-mount", "", "The Weka filesystem mount point on the host")
	dockerLaunchCmd.PersistentFlags().Float64("gpu-mem-util", 0, "GPU memory utilization for vLLM")
	dockerLaunchCmd.PersistentFlags().Int("max-sequences", 0, "The maximum number of sequences")
	dockerLaunchCmd.PersistentFlags().Int("max-model-len", 0, "The maximum model length")
	dockerLaunchCmd.PersistentFlags().Int("max-num-batched-tokens", 0, "The maximum number of batched tokens")
	dockerLaunchCmd.PersistentFlags().Int("port", 0, "The port for the vLLM API server")

	// Add GPU allocation flags
	dockerLaunchCmd.PersistentFlags().String("gpu-slots", "", "Comma-separated list of GPU IDs to use (e.g., '0,1,2,3')")
	dockerLaunchCmd.PersistentFlags().Int("tensor-parallel-size", 0, "Number of GPUs to use for tensor parallelism (used when --gpu-slots is not specified)")

	// Add Docker configuration flags
	dockerLaunchCmd.PersistentFlags().String("docker-image", "", "Docker image to use for the vLLM container (defaults to sdimitro509/amg:v<amgctl-version>, auto-pulled if needed)")
	dockerLaunchCmd.PersistentFlags().Bool("dry-run", false, "Print the Docker command that would be executed without actually running it")

	// Add LMCache configuration flags
	// Note: No default values - these are passed through to 'amgctl host launch' which defines the defaults
	dockerLaunchCmd.PersistentFlags().String("lmcache-weka-path", "", "Path for the cache within the Weka mount")
	dockerLaunchCmd.PersistentFlags().Int("lmcache-chunk-size", 0, "LMCache chunk size")
	dockerLaunchCmd.PersistentFlags().Int("lmcache-gds-threads", 0, "LMCache GDS threads")
	dockerLaunchCmd.PersistentFlags().String("lmcache-cufile-buffer-size", "", "LMCache cuFile buffer size")
	dockerLaunchCmd.PersistentFlags().Bool("lmcache-local-cpu", false, "Enable LMCache local CPU processing")
	dockerLaunchCmd.PersistentFlags().Bool("lmcache-save-decode-cache", false, "Enable LMCache decode cache saving")

	// Add Hugging Face configuration flags
	dockerLaunchCmd.PersistentFlags().String("hf-home", "", "Hugging Face cache directory path")

	// Add Prometheus configuration flags
	dockerLaunchCmd.PersistentFlags().String("prometheus-multiproc-dir", "", "Prometheus multiprocess directory path")

	// Add vLLM configuration flags
	dockerLaunchCmd.PersistentFlags().Bool("no-enable-prefix-caching", false, "Disable vLLM prefix caching")
	dockerLaunchCmd.PersistentFlags().Bool("skip-safefasttensors", false, "Skip adding USE_FASTSAFETENSOR=true env var and --load-format fastsafetensors argument")

	// Add flag to skip default cufile configuration
	dockerLaunchCmd.PersistentFlags().Bool("skip-cufile-configure", false, "Skip automatic generation of cufile.json (cufile.json is generated by default)")

	// Add escape hatch flags for advanced customization
	dockerLaunchCmd.PersistentFlags().StringSlice("docker-arg", []string{}, "Additional arguments to pass to docker run command (repeatable)")
	dockerLaunchCmd.PersistentFlags().StringSlice("vllm-arg", []string{}, "Additional arguments to pass to vllm serve command (repeatable)")
	dockerLaunchCmd.PersistentFlags().StringSlice("vllm-env", []string{}, "Additional environment variables for vllm container in KEY=VALUE format (repeatable)")

	// Bind flags to Viper for configuration management
	// Note: viper.BindPFlag errors are typically only due to programming errors (nil flags)
	// and are safe to ignore in this context as flags are defined above
	_ = viper.BindPFlag("weka-mount", dockerLaunchCmd.PersistentFlags().Lookup("weka-mount"))
	_ = viper.BindPFlag("gpu-mem-util", dockerLaunchCmd.PersistentFlags().Lookup("gpu-mem-util"))
	_ = viper.BindPFlag("max-sequences", dockerLaunchCmd.PersistentFlags().Lookup("max-sequences"))
	_ = viper.BindPFlag("max-model-len", dockerLaunchCmd.PersistentFlags().Lookup("max-model-len"))
	_ = viper.BindPFlag("max-num-batched-tokens", dockerLaunchCmd.PersistentFlags().Lookup("max-num-batched-tokens"))
	_ = viper.BindPFlag("port", dockerLaunchCmd.PersistentFlags().Lookup("port"))
	_ = viper.BindPFlag("gpu-slots", dockerLaunchCmd.PersistentFlags().Lookup("gpu-slots"))
	_ = viper.BindPFlag("tensor-parallel-size", dockerLaunchCmd.PersistentFlags().Lookup("tensor-parallel-size"))
	_ = viper.BindPFlag("docker-image", dockerLaunchCmd.PersistentFlags().Lookup("docker-image"))
	_ = viper.BindPFlag("dry-run", dockerLaunchCmd.PersistentFlags().Lookup("dry-run"))
	_ = viper.BindPFlag("lmcache-weka-path", dockerLaunchCmd.PersistentFlags().Lookup("lmcache-weka-path"))
	_ = viper.BindPFlag("lmcache-chunk-size", dockerLaunchCmd.PersistentFlags().Lookup("lmcache-chunk-size"))
	_ = viper.BindPFlag("lmcache-gds-threads", dockerLaunchCmd.PersistentFlags().Lookup("lmcache-gds-threads"))
	_ = viper.BindPFlag("lmcache-cufile-buffer-size", dockerLaunchCmd.PersistentFlags().Lookup("lmcache-cufile-buffer-size"))
	_ = viper.BindPFlag("lmcache-local-cpu", dockerLaunchCmd.PersistentFlags().Lookup("lmcache-local-cpu"))
	_ = viper.BindPFlag("lmcache-save-decode-cache", dockerLaunchCmd.PersistentFlags().Lookup("lmcache-save-decode-cache"))
	_ = viper.BindPFlag("hf-home", dockerLaunchCmd.PersistentFlags().Lookup("hf-home"))
	_ = viper.BindPFlag("prometheus-multiproc-dir", dockerLaunchCmd.PersistentFlags().Lookup("prometheus-multiproc-dir"))
	_ = viper.BindPFlag("no-enable-prefix-caching", dockerLaunchCmd.PersistentFlags().Lookup("no-enable-prefix-caching"))
	_ = viper.BindPFlag("skip-safefasttensors", dockerLaunchCmd.PersistentFlags().Lookup("skip-safefasttensors"))
	_ = viper.BindPFlag("skip-cufile-configure", dockerLaunchCmd.PersistentFlags().Lookup("skip-cufile-configure"))
	_ = viper.BindPFlag("docker-arg", dockerLaunchCmd.PersistentFlags().Lookup("docker-arg"))
	_ = viper.BindPFlag("vllm-arg", dockerLaunchCmd.PersistentFlags().Lookup("vllm-arg"))
	_ = viper.BindPFlag("vllm-env", dockerLaunchCmd.PersistentFlags().Lookup("vllm-env"))
}

// performPreflightChecks validates system requirements and configuration before execution
func performDockerPreflightChecks() error {
	// Check if docker command exists in PATH
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker command not found in PATH. Please install Docker and ensure it's available in your system PATH")
	}

	// Check if nvidia-ctk command exists in PATH
	if _, err := exec.LookPath("nvidia-ctk"); err != nil {
		return fmt.Errorf("nvidia-ctk command not found in PATH. Please install NVIDIA Container Toolkit and ensure it's available in your system PATH")
	}

	// Check if nvidia-smi command exists in PATH
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return fmt.Errorf("nvidia-smi command not found in PATH. Please install NVIDIA drivers and ensure nvidia-smi is available in your system PATH")
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

	// Check nvidia_peermem kernel module
	if err := checkNvidiaPeermemModule(); err != nil {
		return fmt.Errorf("nvidia_peermem module check failed: %w", err)
	}

	return nil
}
