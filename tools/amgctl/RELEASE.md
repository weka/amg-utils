# Release Process for amgctl

This document describes how to create releases for amgctl and test the self-update functionality.

## Prerequisites

1. Ensure you have push access to the repository
2. The GitHub Actions workflow (`.github/workflows/release.yml`) is in place
3. Your local repository is up to date

## Creating a Release

### 1. Update Version

First, update the version in `main.go`:

```go
const version = "0.2.0"  // Update this
```

### 2. Commit and Tag

```bash
git add .
git commit -m "Release v0.2.0"
git tag v0.2.0
git push origin main
git push origin v0.2.0
```

### 3. Automatic Build

The GitHub Actions workflow will automatically:
- Build binaries for all platforms (linux, darwin, windows; amd64, arm64)
- Generate SHA256 checksums
- Create a GitHub release with all artifacts

## Testing the Update Process

### 1. Manual Testing (Before Publishing)

You can test the release build process locally:

```bash
cd tools/amgctl
make build-release
```

This creates:
- `amgctl-linux-amd64`
- `amgctl-linux-arm64`
- `amgctl-darwin-amd64`
- `amgctl-darwin-arm64`
- `amgctl-windows-amd64.exe`
- `checksums.txt`

### 2. Testing Self-Update

Once you have a release published on GitHub:

```bash
# Check current version
./amgctl --version

# Test update (will only update if there's a newer version)
./amgctl update

# Force update (useful for testing)
./amgctl update --force

# Include pre-releases
./amgctl update --prerelease
```

### 3. Verify Update

After updating:

```bash
# Check new version
./amgctl --version

# Verify functionality
./amgctl --help
```

## Troubleshooting

### Update Command Shows "No releases found"

This happens when:
1. No releases are published on GitHub yet
2. The repository name in `update.go` doesn't match your actual repository
3. Network connectivity issues

### Binary Download Fails

Check:
1. The asset naming in the GitHub release matches what the update command expects
2. The repository is public or you have proper access
3. GitHub releases API is accessible

### Update Fails to Replace Binary

This can happen due to:
1. Permission issues (binary in use, insufficient permissions)
2. Platform-specific file locking
3. Antivirus software blocking the replacement

The update command includes atomic replacement logic and should handle most cases gracefully.

## Release Notes

When creating releases, consider including:
- New features added
- Bug fixes
- Breaking changes
- Installation/update instructions
- Platform-specific notes

The GitHub Actions workflow automatically generates basic release notes, but you can edit them after the release is created for better documentation.
