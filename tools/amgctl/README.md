# amgctl - AMG Scripts Control Tool

`amgctl` is a command line interface for managing Weka AMG (Augmented Memory Grid) environments. It provides tools for setting up, managing, and monitoring AMG environments across Linux, macOS, and Windows platforms.

## Version

Current version: **0.1.2**

## Features

- **Host Environment Management**: Set up, monitor, and clean up AMG environments
- **Cross-Platform Support**: Works on Linux, macOS, and Windows (with platform-specific implementations)
- **Self-Update**: Automatically update to the latest version from GitHub releases
- **Docker Integration**: Manage Docker containers and resources (placeholder for future implementation)
- **Automated Setup**: Replicates the functionality of `setup_lmcache_stable.sh` script

## Prerequisites

### Linux (Fully Implemented)
- Go 1.19 or later
- UV (https://docs.astral.sh/uv/getting-started/installation/)
- Git
- SSH keys configured for GitHub access

### macOS (Placeholder Implementation)
- Go 1.19 or later
- Homebrew (recommended)
- UV (https://docs.astral.sh/uv/getting-started/installation/)
- Git

### Windows (Placeholder Implementation)
- Go 1.19 or later
- Git for Windows
- UV (https://docs.astral.sh/uv/getting-started/installation/)

## Installation

### Building from Source

1. Clone the repository:
```bash
cd /path/to/amg-scripts/tools/amgctl
```

2. Initialize Go module (if not already done):
```bash
go mod init github.com/weka/amg-scripts/tools/amgctl
```

3. Install dependencies:
```bash
go get github.com/spf13/cobra@latest
```

4. Install Git hooks (recommended for development):
```bash
make install-hooks
```

5. Build the binary:
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
Display the current status of the AMG environment (placeholder):

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

### Docker Management (Placeholder)

#### Get Docker Resources
Retrieve information about Docker containers and images (placeholder):

```bash
./amgctl docker get
```

## Configuration

The tool uses the following default configuration values:

- **UV Virtual Environment Name**: `amg_stable`
- **Repository URL**: `git@github.com:weka/weka-LMCache.git`
- **Repository Name**: `LMCache`
- **Default Branch**: `dev` (can be overridden with `--lmcache-branch` or `--lmcache-commit`)
- **Base Path**: `~/amg_stable`
- **vLLM Version**: `0.9.2` (configurable via `--vllm-version` flag)

These values are currently hardcoded but may be made configurable in future versions.

## Platform-Specific Notes

### Linux (Fully Functional)
- All features are implemented and tested
- Requires standard Linux development tools
- Uses system UV installation

### macOS (Placeholder)
- Basic structure implemented
- Platform-specific optimizations planned
- Homebrew integration planned

### Windows (Placeholder)
- Basic structure implemented
- PowerShell/cmd compatibility planned
- Windows-specific path handling planned

## Error Handling

The tool provides comprehensive error handling:
- Command existence checks before execution
- Graceful failure with informative error messages
- Warning messages for non-critical failures
- Cross-platform error handling

## Development

### Adding New Commands

1. Create a new Go file for your command group (e.g., `newcmd.go`)
2. Define the command structure using cobra
3. Add the command to the root command in `main.go`
4. Implement platform-specific logic as needed

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
git commit -m "Fix bug in host setup"

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
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. **Setup**: Install Git hooks with `make install-hooks`
4. Make your changes
5. **Automatic**: Linting runs on commit via Git hooks
   - Or manually run: `make lint-all`
   - Fix issues with: `make fix`
6. Test on your platform
7. Submit a pull request

**Note**: Git hooks automatically run when you commit changes to amgctl. Install them once with `make install-hooks`.

## License

Licensed under the Apache License, Version 2.0. See the LICENSE file in the root directory for details.

## Support

For issues and questions:
1. Check existing issues in the repository
2. Create a new issue with detailed information
3. Include platform information and error logs
