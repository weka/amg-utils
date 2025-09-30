#!/bin/bash

# AMG Docker Image Build and Push Script
# This script builds and pushes the AMG Docker image to Docker Hub
# Usage: ./docker/scripts/build-and-push-docker.sh [version]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }

# Function to show usage
show_usage() {
    echo "Usage: $0 [version]"
    echo ""
    echo "Arguments:"
    echo "  version    Version tag for the Docker image (e.g., v0.1.19)"
    echo "             If not provided, will attempt to detect from git tag"
    echo ""
    echo "Environment Variables:"
    echo "  DOCKERHUB_USERNAME  Docker Hub username (required)"
    echo "  DOCKERHUB_TOKEN     Docker Hub access token (required)"
    echo ""
    echo "Examples:"
    echo "  $0 v0.1.19"
    echo "  $0"
    echo ""
}

# Check if Docker is installed and running
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running"
        exit 1
    fi
}

# Get version from argument or git tag
get_version() {
    local version="$1"
    
    if [ -n "$version" ]; then
        echo "$version"
        return
    fi
    
    # Try to get version from git tag
    if git describe --tags --exact-match HEAD 2>/dev/null; then
        return
    fi
    
    # Try to get latest tag
    local latest_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    if [ -n "$latest_tag" ]; then
        print_warning "Not on a tagged commit. Latest tag is: $latest_tag"
        read -p "Do you want to use $latest_tag? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo "$latest_tag"
            return
        fi
    fi
    
    print_error "Could not determine version. Please provide it as an argument."
    show_usage
    exit 1
}

# Check required environment variables
check_env_vars() {
    if [ -z "$DOCKERHUB_USERNAME" ]; then
        print_error "DOCKERHUB_USERNAME environment variable is required"
        echo "Export it with: export DOCKERHUB_USERNAME=your_username"
        exit 1
    fi
    
    if [ -z "$DOCKERHUB_TOKEN" ]; then
        print_error "DOCKERHUB_TOKEN environment variable is required"
        echo "Export it with: export DOCKERHUB_TOKEN=your_token"
        echo "You can create a token at: https://hub.docker.com/settings/security"
        exit 1
    fi
}

# Login to Docker Hub
docker_login() {
    print_info "Logging in to Docker Hub as $DOCKERHUB_USERNAME..."
    echo "$DOCKERHUB_TOKEN" | docker login -u "$DOCKERHUB_USERNAME" --password-stdin
    print_success "Successfully logged in to Docker Hub"
}

# Build and push Docker images (both variants)
build_and_push() {
    local version="$1"
    local docker_context="./docker"
    local base_image_name="sdimitro509/amg"
    
    print_info "Building Docker images for version: $version"
    print_info "Docker context: $docker_context"
    
    # Check if docker context directory exists
    if [ ! -d "$docker_context" ]; then
        print_error "Docker context directory '$docker_context' not found"
        print_error "Please run this script from the repository root"
        exit 1
    fi
    
    # Set up Docker Buildx if not already done
    print_info "Setting up Docker Buildx..."
    docker buildx create --use --name amg-builder --driver docker-container 2>/dev/null || true
    docker buildx inspect --bootstrap >/dev/null 2>&1 || {
        print_warning "Docker Buildx not available, falling back to regular docker build"
        USE_BUILDX=false
    }
    
    # Build arguments
    local build_args="--build-arg AMG_UTILS_VERSION=$version"
    
    # Build and push POC variant (full setup)
    local poc_image_name="${base_image_name}-poc"
    local poc_tags="-t $poc_image_name:$version -t $poc_image_name:latest"
    
    print_info "Building and pushing POC variant (full setup with amgctl host setup)..."
    if [ "${USE_BUILDX:-true}" = "true" ]; then
        docker buildx build \
            --platform linux/amd64 \
            --push \
            --target poc \
            $build_args \
            $poc_tags \
            "$docker_context"
    else
        docker build \
            --target poc \
            $build_args \
            $poc_tags \
            "$docker_context"
        
        docker push "$poc_image_name:$version"
        docker push "$poc_image_name:latest"
    fi
    
    print_success "Successfully built and pushed POC images:"
    print_success "  - $poc_image_name:$version"
    print_success "  - $poc_image_name:latest"
    
    # Build and push Vanilla variant (base only)
    local vanilla_image_name="${base_image_name}-vanilla"
    local vanilla_tags="-t $vanilla_image_name:$version -t $vanilla_image_name:latest"
    
    print_info "Building and pushing Vanilla variant (base with amgctl only)..."
    if [ "${USE_BUILDX:-true}" = "true" ]; then
        docker buildx build \
            --platform linux/amd64 \
            --push \
            --target vanilla \
            $build_args \
            $vanilla_tags \
            "$docker_context"
    else
        docker build \
            --target vanilla \
            $build_args \
            $vanilla_tags \
            "$docker_context"
        
        docker push "$vanilla_image_name:$version"
        docker push "$vanilla_image_name:latest"
    fi
    
    print_success "Successfully built and pushed Vanilla images:"
    print_success "  - $vanilla_image_name:$version"
    print_success "  - $vanilla_image_name:latest"
    
    echo ""
    print_success "All images built and pushed successfully!"
}

# Cleanup function
cleanup() {
    if [ "${USE_BUILDX:-true}" = "true" ]; then
        print_info "Cleaning up Docker Buildx builder..."
        docker buildx rm amg-builder 2>/dev/null || true
    fi
}

# Main execution
main() {
    # Parse arguments
    if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
        show_usage
        exit 0
    fi
    
    # Set up cleanup on exit
    trap cleanup EXIT
    
    print_info "AMG Docker Image Build and Push Script"
    print_info "====================================="
    
    # Pre-flight checks
    check_docker
    check_env_vars
    
    # Get version
    local version=$(get_version "$1")
    print_info "Using version: $version"
    
    # Change to repository root if not already there
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local repo_root="$(cd "$script_dir/../.." && pwd)"
    cd "$repo_root"
    
    print_info "Working directory: $(pwd)"
    
    # Confirm before proceeding
    echo
    print_warning "About to build and push Docker images:"
    echo "  Version: $version"
    echo "  POC Images: sdimitro509/amg-poc:$version, sdimitro509/amg-poc:latest"
    echo "  Vanilla Images: sdimitro509/amg-vanilla:$version, sdimitro509/amg-vanilla:latest"
    echo "  Registry: Docker Hub"
    echo
    read -p "Do you want to continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Aborted by user"
        exit 0
    fi
    
    # Execute build and push
    docker_login
    build_and_push "$version"
    
    print_success "Docker images build and push completed successfully!"
    print_info "You can now run:"
    print_info "  docker pull sdimitro509/amg-poc:$version"
    print_info "  docker pull sdimitro509/amg-vanilla:$version"
}

# Run main function with all arguments
main "$@"
