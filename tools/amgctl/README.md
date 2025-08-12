# amgctl - AMG Scripts Control Tool

`amgctl` is a command line interface for managing AMG (AI Model Gateway) environments. It provides tools for setting up, managing, and monitoring AMG environments across Linux, macOS, and Windows platforms.

## Version

Current version: **0.1.0**

## Features

- **Host Environment Management**: Set up, monitor, and clean up AMG environments
- **Cross-Platform Support**: Works on Linux, macOS, and Windows (with platform-specific implementations)
- **Docker Integration**: Manage Docker containers and resources (placeholder for future implementation)
- **Automated Setup**: Replicates the functionality of `setup_lmcache_stable.sh` script

## Prerequisites

### Linux (Fully Implemented)
- Go 1.19 or later
- Conda (Anaconda or Miniconda)
- Git
- SSH keys configured for GitHub access

### macOS (Placeholder Implementation)
- Go 1.19 or later
- Homebrew (recommended)
- Conda (Anaconda or Miniconda)
- Git

### Windows (Placeholder Implementation)
- Go 1.19 or later
- Git for Windows
- Conda (Anaconda or Miniconda)

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

4. Build the binary:
```bash
go build -o amgctl .
```

### Cross-Platform Builds

Build for different platforms:

```bash
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
Set up the complete AMG environment including conda environment, repository cloning, and dependency installation:

```bash
./amgctl host setup
```

This command performs the following actions:
1. **Initial Checks**: Verifies that conda and git are installed
2. **Conda Environment**: Creates `amg_stable` environment with Python 3.12
3. **Package Installation**: Installs required Python packages including:
   - vLLM wheel from specific commit
   - py-spy, scalene, pyinstrument, line_profiler
4. **Repository Management**: 
   - Clones `weka-LMCache` repository to `~/amg-stable/LMCache`
   - Checks out specific commit: `c231e2285ee61a0cbc878d51ed2e7236ac7c0b5d`
5. **Dependencies**: Installs repository dependencies from requirements files
6. **Editable Install**: Installs the repository in editable mode
7. **Hot-patches**: Downgrades transformers package for compatibility

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
- Removes the `amg_stable` conda environment
- Deletes the `~/amg-stable` directory and all contents

### Docker Management (Placeholder)

#### Get Docker Resources
Retrieve information about Docker containers and images (placeholder):

```bash
./amgctl docker get
```

## Configuration

The tool uses the following default configuration values:

- **Conda Environment Name**: `amg_stable`
- **Repository URL**: `git@github.com:weka/weka-LMCache.git`
- **Repository Name**: `LMCache`
- **Target Commit**: `c231e2285ee61a0cbc878d51ed2e7236ac7c0b5d`
- **Base Path**: `~/amg-stable`
- **vLLM Commit**: `b6553be1bc75f046b00046a4ad7576364d03c835`

These values are currently hardcoded but may be made configurable in future versions.

## Platform-Specific Notes

### Linux (Fully Functional)
- All features are implemented and tested
- Requires standard Linux development tools
- Uses system conda installation

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

### Testing

```bash
# Run tests (when available)
go test ./...

# Build and test manually
go build -o amgctl .
./amgctl --help
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test on your platform
5. Submit a pull request

## License

Licensed under the Apache License, Version 2.0. See the LICENSE file in the root directory for details.

## Support

For issues and questions:
1. Check existing issues in the repository
2. Create a new issue with detailed information
3. Include platform information and error logs

## Roadmap

- **v0.2.0**: Complete macOS and Windows implementations
- **v0.3.0**: Docker command implementations
- **v0.4.0**: Configuration file support
- **v0.5.0**: Enhanced status reporting and monitoring
