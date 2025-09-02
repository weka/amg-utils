# AMG-QAD (AMG Quality Assurance Daemon)

AMG-QAD is a daemon service that runs scheduled quality assurance tests for AMG environments. It provides automated testing, result storage, and a web dashboard to view test results.

## Features

- **Scheduled Testing**: Runs QA tests at a specified time each day
- **Web Dashboard**: View the last 10 test runs with pass/fail status and timestamps
- **Result Storage**: Persistent storage of test results with timestamps, duration, and logs
- **RESTful API**: JSON API endpoint for programmatic access to results
- **Configurable**: YAML-based configuration with sensible defaults

## Installation

### Build from Source

```bash
# Clone the repository (if not already done)
cd tools/amg-qad

# Install Git hooks (recommended for development)
make install-hooks

# Install dependencies and build
make all

# Install binary (optional)
make install
```

### Pre-built Binaries

Download the latest release from GitHub:
- **Linux (x64)**: `amg-qad-linux-amd64`

Or build for your platform:
```bash
# Build for current platform
make build

# Build for Linux AMD64
make build-linux-amd64
```

## Configuration

Create a configuration file at `~/.config/amg-qad.yaml`:

```yaml
# Time of day to run tests (24-hour format HH:MM)
test_time: "02:00"

# Port for the web dashboard
web_port: 8080

# Path to store test results
results_path: "./results"
```

You can also use the provided example:

```bash
cp amg-qad.yaml.example ~/.config/amg-qad.yaml
```

Or generate a sample configuration:

```bash
make config
```

## Usage

### Start the Daemon

```bash
# Start with default configuration
./amg-qad daemon

# Start with custom configuration file
./amg-qad daemon --config /path/to/config.yaml
```

### Environment Variables

Configuration can also be set via environment variables with the `AMG_QAD_` prefix:

```bash
export AMG_QAD_TEST_TIME="03:00"
export AMG_QAD_WEB_PORT=9090
export AMG_QAD_RESULTS_PATH="/var/lib/amg-qad/results"
./amg-qad daemon
```

## Web Dashboard

Once the daemon is running, access the web dashboard at:
- Default: http://localhost:8080
- Custom port: http://localhost:YOUR_PORT

The dashboard shows:
- Test statistics (total, passed, failed, success rate)
- Last 10 test runs with status and timing information
- Auto-refresh every 30 seconds

## API Endpoints

### Get Test Results

```bash
# Get last 10 results (default)
curl http://localhost:8080/api/results

# Get specific number of results (max 100)
curl http://localhost:8080/api/results?limit=25
```

Response format:
```json
[
  {
    "timestamp": "2024-01-15T02:00:05Z",
    "status": "passed",
    "duration": "2.5s",
    "parameters": "placeholder_test"
  }
]
```

## Test Implementation

Currently, AMG-QAD includes a placeholder test that:
- Simulates test execution with random duration (1-3 seconds)
- Has a 90% success rate (10% simulated failures)
- Records execution time and basic logs

### Future Test Integration

The architecture is designed to easily integrate real AMG tests:

1. Download `amgctl` from the repository
2. Execute various `amgctl` commands
3. Analyze command output and exit codes
4. Record detailed logs for failures

## File Structure

```
amg-qad/
├── main.go              # Application entry point
├── cmd/                 # CLI commands
│   ├── root.go          # Root command and config
│   └── daemon.go        # Daemon command
├── internal/            # Internal packages
│   ├── config/          # Configuration handling
│   ├── scheduler/       # Test scheduling and execution
│   ├── storage/         # Result storage
│   └── web/             # Web dashboard and API
├── results/             # Default results directory
├── Makefile             # Build automation
├── amg-qad.yaml.example # Example configuration
└── README.md            # This file
```

## Development

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

- **Automatic**: Runs when you commit changes to `tools/amg-qad/`
- **Smart**: Only runs checks when amg-qad files are modified  
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

### Building and Testing

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

# Format code (modifies files)
make fmt

# Run tests
make test

# Run tests with coverage
make test-coverage

# Clean build artifacts
make clean

# Install Git hooks for automatic linting on commit
make install-hooks

# Show all available targets
make help
```

### Adding New Tests

To add new test implementations:

1. Implement the `TestRunner` interface in `internal/scheduler/test.go`
2. Update the scheduler to use your new test runner
3. Configure test parameters via configuration file

## Contributing

1. Fork the repository
2. Create a feature branch
3. **Setup**: Install Git hooks with `make install-hooks`
4. Make your changes
5. **Automatic**: Linting runs on commit via Git hooks
   
   - Fix issues with: `make fix`
6. Test your changes
7. Submit a pull request

**Note**: Git hooks automatically run when you commit changes to amg-qad. Install them once with `make install-hooks`.

## Configuration Reference

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `test_time` | string | "02:00" | Daily test execution time (HH:MM) |
| `web_port` | int | 8080 | Web dashboard and API port |
| `results_path` | string | "./results" | Directory for storing test results |

## Releases

AMG-QAD releases are automatically created when tags are pushed to the repository. Each release includes:

- **AMD64 Linux binary**: `amg-qad-linux-amd64`
- **SHA256 checksums**: `checksums.txt` for verification
- **Automated release notes**: Generated from git commits
- **Installation instructions**: Quick start guide

### Creating a Release

```bash
# Tag a new version
git tag v0.2.0

# Push the tag to trigger release
git push origin v0.2.0
```

## Troubleshooting

### Common Issues

1. **Port already in use**: Change `web_port` in configuration
2. **Permission denied on results_path**: Ensure directory is writable
3. **Configuration not found**: Check file path and permissions

### Logs

The daemon logs important events to stdout/stderr:
- Startup and configuration loading
- Test scheduling and execution
- Web server status
- Error conditions

### Results Storage

Results are stored in JSON Lines format in `results_path/results.jsonl`. Each line contains:
```json
{
  "timestamp": "2024-01-15T02:00:05Z",
  "passed": true,
  "duration": "2.5s", 
  "parameters": "placeholder_test",
  "logs": "Test completed successfully..."
}
```
