# AMG Build Scripts

This directory contains scripts for building and publishing AMG Docker images manually.

## Scripts

### 1. `build-docker-local.sh` - Local Development

Builds Docker images locally for testing and development.

**Usage:**
```bash
# Build with 'dev' tag (default)
./docker/scripts/build-docker-local.sh

# Build with specific version
./docker/scripts/build-docker-local.sh v0.1.9
```

**Features:**
- No Docker Hub credentials required
- Builds locally for testing
- Uses 'dev' tag by default
- Creates additional `local-<version>` tag for non-dev builds

### 2. `build-and-push-docker.sh` - Production Release

Builds and pushes Docker images to Docker Hub for production release.

**Usage:**
```bash
# Push current git tag
./docker/scripts/build-and-push-docker.sh

# Push specific version
./docker/scripts/build-and-push-docker.sh v0.1.9
```

**Prerequisites:**
Set these environment variables:
```bash
export DOCKERHUB_USERNAME=your_username
export DOCKERHUB_TOKEN=your_access_token
```

**Features:**
- Automatic version detection from git tags
- Pushes to `sdimitro509/amg:version` and `sdimitro509/amg:latest`
- Uses Docker Buildx for multi-platform builds
- Interactive confirmation before pushing
- Comprehensive error checking

## Docker Hub Token Setup

1. Go to [Docker Hub Settings > Security](https://hub.docker.com/settings/security)
2. Click "New Access Token"
3. Give it a descriptive name (e.g., "AMG Build Script")
4. Copy the generated token
5. Export it in your shell:
   ```bash
   export DOCKERHUB_TOKEN=your_token_here
   ```

## Examples

### Development Workflow
```bash
# Build and test locally
./docker/scripts/build-docker-local.sh

# Test the image
docker run -it --rm --gpus all --runtime nvidia sdimitro509/amg:dev

# When ready for release, tag and push
git tag v0.1.9
git push origin v0.1.9
./docker/scripts/build-and-push-docker.sh
```

### Quick Release
```bash
# Set up credentials (one time)
export DOCKERHUB_USERNAME=sdimitro509
export DOCKERHUB_TOKEN=your_token

# Build and push latest tag
./docker/scripts/build-and-push-docker.sh
```

## Migration from GitHub Actions

Previously, Docker images were built and pushed automatically via GitHub Actions on tag pushes. This has been replaced with these manual scripts for better control over the release process.

The GitHub Actions workflow now only:
- Builds binaries
- Creates GitHub releases
- Uploads binary artifacts

Docker images must be built and pushed manually using these scripts.

## Troubleshooting

### Docker Buildx Issues
If you encounter issues with Docker Buildx, the script will fall back to regular `docker build` and `docker push` commands.

### Permission Issues
Ensure the scripts are executable:
```bash
chmod +x docker/scripts/*.sh
```

### Network Issues
If pushing fails due to network issues, you can retry just the push step by running the script again with the same version.

### Version Detection
The script tries to detect versions in this order:
1. Command line argument
2. Current git tag (if on exact commit)
3. Latest git tag (with confirmation)
4. Prompts for manual input
