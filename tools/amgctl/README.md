# amgctl - AMG Utils Control Tool

`amgctl` is a command line interface for managing Weka AMG (Augmented Memory Grid) environments. It provides tools for setting up, managing, and monitoring AMG environments with advanced hardware discovery capabilities for HPC workloads.

## Version

Current version: **0.1.6**

## Features

- **Advanced Hardware Discovery**: Automatic detection of NVIDIA GPUs and InfiniBand devices
- **Container Orchestration**: Launch AMG containers with optimized hardware configurations
- **Host Environment Management**: Set up, monitor, and clean up AMG environments
- **Cross-Platform Support**: Works on Linux, macOS, and Windows (with platform-specific implementations)
- **Self-Update**: Automatically update to the latest version from GitHub releases
- **Docker Integration**: Manage Docker containers with automatic device flag generation
- **Automated Setup**: Replicates the functionality of `setup_lmcache_stable.sh` script

## Prerequisites

### Linux (Fully Implemented)
- Go 1.23 or later
- UV (https://docs.astral.sh/uv/getting-started/installation/)
- Git
- SSH keys configured for GitHub access
- NVIDIA drivers (for GPU detection)
- InfiniBand drivers (for InfiniBand detection)

### macOS (Placeholder Implementation)
- Go 1.23 or later
- Homebrew (recommended)
- UV (https://docs.astral.sh/uv/getting-started/installation/)
- Git

### Windows (Placeholder Implementation)
- Go 1.23 or later
- Git for Windows
- UV (https://docs.astral.sh/uv/getting-started/installation/)

## Installation

### Building from Source

1. Clone the repository:
```bash
cd /path/to/amg-utils/tools/amgctl
```

2. Install dependencies:
```bash
go mod download
```

3. Install Git hooks (recommended for development):
```bash
make install-hooks
```

4. Build the binary:
```bash
go build -o amgctl .
# Or using Make
make build
```

### Cross-Platform Builds

Build for different platforms:

```bash
# Using Make (recommended)
make build-all

# Or manually with Go
# Linux
GOOS=linux GOARCH=amd64 go build -o amgctl-linux-amd64 .

# macOS
GOOS=darwin GOARCH=amd64 go build -o amgctl-darwin-amd64 .
GOOS=darwin GOARCH=arm64 go build -o amgctl-darwin-arm64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o amgctl-windows-amd64.exe .
```

## Usage

### Basic Commands

```bash
# Show help
./amgctl --help

# Show version
./amgctl --version
```

### Docker Container Management

#### Launch AMG Container
Launch an AMG container with the specified model and comprehensive hardware optimization:

```bash
# Basic usage
./amgctl docker launch meta-llama/Llama-2-7b-chat-hf

# Advanced configuration
./amgctl docker launch microsoft/DialoGPT-medium \
  --gpu-mem-util 0.8 \
  --port 8080 \
  --max-sequences 512 \
  --weka-mount /custom/weka/path \
  --lmcache-chunk-size 512
```

#### Launch Command Options

The `docker launch` command supports comprehensive configuration options:

**Required Arguments:**
- `model_identifier`: The model to deploy (e.g., `meta-llama/Llama-2-7b-chat-hf`)

**vLLM Configuration Flags:**
- `--weka-mount`: Weka filesystem mount point (default: `/mnt/weka`)
- `--gpu-mem-util`: GPU memory utilization for vLLM (default: `0.9`)
- `--max-sequences`: Maximum number of sequences (default: `256`)
- `--max-model-len`: Maximum model length (default: `16384`)
- `--port`: Port for the vLLM API server (default: `8000`)

**LMCache Configuration Flags:**
- `--lmcache-path`: Path for cache within Weka mount (default: `/mnt/weka/cache`)
- `--lmcache-chunk-size`: LMCache chunk size (default: `256`)
- `--lmcache-gds-threads`: LMCache GDS threads (default: `32`)

**Hardware Discovery Features:**
- **Automatic GPU Detection**: Discovers all NVIDIA GPUs with detailed information
- **InfiniBand Discovery**: Generates complete Docker device flags for all IB devices
- **Device Flag Generation**: Ready-to-use Docker flags for container deployment

**Examples:**
```bash
# Launch with custom GPU memory utilization
./amgctl docker launch openai/gpt-3.5-turbo --gpu-mem-util 0.7

# Launch with custom port and LMCache settings
./amgctl docker launch huggingface/CodeBERTa-small-v1 \
  --port 9000 \
  --lmcache-chunk-size 512 \
  --lmcache-gds-threads 64

# Launch with custom Weka mount point
./amgctl docker launch microsoft/DialoGPT-large \
  --weka-mount /shared/storage \
  --lmcache-path /shared/storage/cache
```

#### Hardware Discovery Output

The launch command automatically detects and displays:

**GPU Information:**
```
Detected 8 NVIDIA GPU(s)
GPU Details:
  GPU 0: NVIDIA H100 80GB HBM3
  GPU 1: NVIDIA H100 80GB HBM3
  ...
```

**InfiniBand Device Flags:**
```
InfiniBand Docker Device Flags:
--device=/dev/infiniband/rdma_cm --device=/dev/infiniband/uverbs0 --device=/dev/infiniband/uverbs1 ...

InfiniBand Device Details:
  InfiniBand Device: mlx5_0
    Board ID: MT_0000000891
    Firmware Version: 28.43.2026
    Ports: 1
      Port 1: State=ACTIVE
```

### Host Environment Management

#### Setup AMG Environment
Set up the complete AMG environment including UV virtual environment, repository cloning, and dependency installation:

```bash
./amgctl host setup

# Install with a different vLLM version
./amgctl host setup --vllm-version 0.10.0

# Other available flags
./amgctl host setup --skip-hotfixes --vllm-version 0.9.1
```

This command performs the following actions:
1. **Initial Checks**: Verifies that uv and git are installed
2. **UV Virtual Environment**: Creates `amg_stable` virtual environment with Python 3.12
3. **Package Installation**: Installs required Python packages including:
   - vLLM version 0.9.2 (configurable via `--vllm-version` flag) with torch dependencies
   - py-spy, scalene, pyinstrument, line_profiler, fastsafetensors
4. **Repository Management**: 
   - Clones `weka-LMCache` repository to `~/amg_stable/LMCache`
   - Checks out and tracks the `dev` branch by default
5. **Dependencies**: Installs repository dependencies from requirements files (with --no-build-isolation)
6. **Editable Install**: Installs the repository in editable mode (with --no-build-isolation)
7. **Hot-patches**: Downgrades transformers package for compatibility

#### Setup Command Options

The `host setup` command supports several flags for customization:

- `--vllm-version`: Specify vLLM version to install (default: "0.9.2")
- `--skip-hotfixes`: Skip applying hotfixes like downgrading transformers
- `--lmcache-repo`: Alternative LMCache repository URL
- `--lmcache-commit`: Specific commit hash for LMCache repository
- `--lmcache-branch`: Branch to follow for LMCache repository (overrides commit)

Examples:
```bash
# Use different vLLM version
./amgctl host setup --vllm-version 0.10.0

# Skip hotfixes and use custom vLLM version
./amgctl host setup --skip-hotfixes --vllm-version 0.9.1

# Use different repository branch
./amgctl host setup --lmcache-branch main --vllm-version 0.9.2
```

#### After Setup: Using the AMG Environment

Once the setup is complete, follow these steps to start using the AMG environment:

1. **Navigate to the AMG environment directory**:
   ```bash
   cd ~/amg_stable
   ```

2. **Activate the virtual environment**:
   ```bash
   source .venv/bin/activate
   ```

3. **Verify activation**: Your shell prompt should now show `(amg)` at the beginning, indicating the environment is active.

4. **Use the AMG tools**: You can now run LMCache and other AMG tools within this environment.

5. **Deactivate when done**: To return to your normal shell environment, run:
   ```bash
   deactivate
   ```

**Important Notes**:
- Always ensure you have deactivated any conda environments before running amgctl host commands
- The environment is located at `~/amg_stable/` with the virtual environment in `~/amg_stable/.venv/`
- The LMCache repository is cloned to `~/amg_stable/LMCache/`

#### Check Environment Status
Display the current status of the AMG environment:

```bash
./amgctl host status
```

#### Clear AMG Environment
Remove all components created by the setup command:

```bash
./amgctl host clear
```

This command:
- Removes the `amg_stable` UV virtual environment
- Deletes the `~/amg_stable` directory and all contents

### Self-Update

#### Update to Latest Version
Update amgctl to the latest version from GitHub releases:

```bash
./amgctl update
```

This command will:
- Check for the latest release on GitHub
- Download the appropriate binary for your platform
- Verify the download integrity using SHA256 checksums
- Replace the current binary atomically with rollback capability

#### Update Options

```bash
# Force update even if already on latest version
./amgctl update --force

# Include pre-release versions
./amgctl update --prerelease
```

## Hardware Discovery

The amgctl tool includes advanced hardware discovery capabilities for HPC workloads:

### NVIDIA GPU Detection

- **Automatic Discovery**: Uses NVIDIA Go bindings (`go-nvml`) for accurate GPU detection
- **Detailed Information**: Displays GPU models, memory, and device count
- **Error Handling**: Graceful handling when NVIDIA drivers are not available

### InfiniBand Discovery

- **Complete Device Scanning**: Discovers all InfiniBand devices using `prometheus/procfs`
- **Docker Integration**: Generates ready-to-use Docker device flags
- **Device Types**: Automatically detects:
  - `uverbs` devices for user-space access
  - `umad` devices for user-space management
  - `issm` devices for subnet management
  - `rdma_cm` device for connection management
- **Port Discovery**: Detects all active ports with state information

### Supported Hardware

- **NVIDIA GPUs**: All CUDA-compatible GPUs (Tesla, Quadro, GeForce)
- **InfiniBand**: Mellanox ConnectX series and compatible adapters
- **RDMA**: Full RDMA/RoCE device support

## Architecture

### Project Structure

```
tools/amgctl/
├── cmd/                    # Cobra command definitions
│   ├── docker.go          # Docker management commands
│   ├── host.go            # Host environment commands
│   ├── launch.go          # Container launch command
│   ├── root.go            # Root command and CLI setup
│   └── update.go          # Self-update functionality
├── internal/              # Internal packages
│   └── hardware/          # Hardware discovery modules
│       ├── gpu.go         # NVIDIA GPU detection
│       └── infiniband.go  # InfiniBand device discovery
├── go.mod                 # Go module definition
├── go.sum                 # Go module checksums
├── main.go               # Application entry point
├── Makefile              # Build and development tasks
└── README.md             # This file
```

### Dependencies

**Core Dependencies:**
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management

**Hardware Discovery Dependencies:**
- `github.com/NVIDIA/go-nvml` - NVIDIA GPU detection
- `github.com/prometheus/procfs` - System information parsing for InfiniBand

## Configuration

The tool supports configuration through:

- **Command-line flags**: Override defaults for specific commands
- **Configuration files**: Viper-based config file support (default: `~/.config/amgctl.yaml`)
- **Environment variables**: Automatic environment variable binding with `AMGCTL_` prefix

### Default Configuration Values

- **UV Virtual Environment Name**: `amg_stable`
- **Repository URL**: `git@github.com:weka/weka-LMCache.git`
- **Repository Name**: `LMCache`
- **Default Branch**: `dev` (can be overridden with `--lmcache-branch` or `--lmcache-commit`)
- **Base Path**: `~/amg_stable`
- **vLLM Version**: `0.9.2` (configurable via `--vllm-version` flag)
- **Docker Configuration**: See launch command options above

## Platform-Specific Notes

### Linux (Fully Functional)
- All features are implemented and tested
- Full hardware discovery support
- Requires standard Linux development tools
- Uses system UV installation

### macOS (Limited Support)
- Basic structure implemented
- Hardware discovery may have limitations
- Platform-specific optimizations planned
- Homebrew integration planned

### Windows (Placeholder)
- Basic structure implemented
- Hardware discovery not implemented
- PowerShell/cmd compatibility planned
- Windows-specific path handling planned

## Error Handling

The tool provides comprehensive error handling:
- Command existence checks before execution
- Graceful failure with informative error messages
- Warning messages for non-critical failures (e.g., missing hardware)
- Cross-platform error handling
- Hardware detection fallbacks

## Development

### Adding New Commands

1. Create a new Go file for your command group (e.g., `newcmd.go`)
2. Define the command structure using cobra
3. Add the command to the root command in `cmd/root.go`
4. Implement platform-specific logic as needed

### Adding Hardware Discovery

1. Create new files in `internal/hardware/`
2. Implement discovery functions following the existing patterns
3. Add integration to the launch command
4. Include appropriate error handling for missing hardware

### Git Hooks Setup

The project includes Git hooks that automatically run linting and formatting checks on commit:

#### Installation

```bash
# Install hooks (run once after cloning)
make install-hooks

# Or manually
./scripts/install-hooks.sh
```

#### Behavior

- **Automatic**: Runs when you commit changes to `tools/amgctl/`
- **Smart**: Only runs checks when amgctl files are modified  
- **Comprehensive**: Runs `make fmt-check`, `make vet`, and `make lint`
- **Helpful**: Provides clear error messages and fix suggestions

```bash
# Normal commit - hook runs automatically
git commit -m "Add new feature"

# Skip hook if needed (not recommended)
git commit --no-verify -m "Emergency commit"

# If hook fails, fix issues and retry
make fix              # Auto-fix formatting and linting
git add .            # Stage the fixes
git commit -m "..."  # Commit again
```

### Make Commands

The project includes a comprehensive Makefile with linting, formatting, and build targets:

```bash
# Build the binary
make build

# Run all linting and formatting checks (recommended for development)
make lint-all

# Fix formatting and auto-fixable linting issues
make fix

# Install golangci-lint if not present
make lint-install

# Individual linting commands
make fmt-check    # Check formatting (read-only)
make vet          # Run go vet
make lint         # Run golangci-lint

# Build for all platforms
make build-all

# Clean build artifacts
make clean

# Install Git hooks for automatic linting on commit
make install-hooks

# Show all available targets
make help
```

### Testing

```bash
# Run tests (when available)
go test ./...
# Or using Make
make test

# Build and test manually
go build -o amgctl .
./amgctl --help
# Or using Make
make build
./amgctl --help

# Test hardware discovery
./amgctl docker launch test-model
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. **Setup**: Install Git hooks with `make install-hooks`
4. Make your changes
5. **Automatic**: Linting runs on commit via Git hooks
   - Or manually run: `make lint-all`
   - Fix issues with: `make fix`
6. Test on your platform (especially hardware discovery features)
7. Submit a pull request

**Note**: Git hooks automatically run when you commit changes to amgctl. Install them once with `make install-hooks`.

## Troubleshooting

### Hardware Discovery Issues

**NVIDIA GPU Detection:**
- Ensure NVIDIA drivers are installed
- Check that `/usr/bin/nvidia-smi` works
- Verify NVML library is available

**InfiniBand Detection:**
- Ensure InfiniBand drivers are installed
- Check that `/sys/class/infiniband` exists
- Verify `/dev/infiniband/` devices are present

**Common Solutions:**
```bash
# Check GPU availability
nvidia-smi

# Check InfiniBand devices
ls /dev/infiniband/

# Test hardware discovery
./amgctl docker launch test-model --help
```

### Build Issues

```bash
# Clean and rebuild
make clean
go mod tidy
make build

# Verify dependencies
go mod verify
```

## License

Licensed under the Apache License, Version 2.0. See the LICENSE file in the root directory for details.

## Support

For issues and questions:
1. Check existing issues in the repository
2. Create a new issue with detailed information
3. Include platform information, hardware details, and error logs
4. For hardware discovery issues, include output of `nvidia-smi` and `ls /dev/infiniband/`
