#!/bin/bash

# AMG Docker Image Local Build Script
# This script builds the AMG Docker image locally for testing
# Usage: ./docker/scripts/build-docker-local.sh [version]

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
    echo "  version    Version tag for the Docker image (e.g., v0.1.8)"
    echo "             If not provided, will use 'dev' as the tag"
    echo ""
    echo "Examples:"
    echo "  $0 v0.1.9           # Build with specific version"
    echo "  $0                  # Build with 'dev' tag"
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

# Get version from argument or use default
get_version() {
    local version="$1"
    
    if [ -n "$version" ]; then
        echo "$version"
        return
    fi
    
    # Use 'dev' as default for local builds
    echo "dev"
}

# Build Docker image locally
build_local() {
    local version="$1"
    local docker_context="./docker"
    local image_name="sdimitro509/amg"
    
    print_info "Building Docker image locally for version: $version"
    print_info "Docker context: $docker_context"
    
    # Check if docker context directory exists
    if [ ! -d "$docker_context" ]; then
        print_error "Docker context directory '$docker_context' not found"
        print_error "Please run this script from the repository root"
        exit 1
    fi
    
    # Build arguments
    local build_args="--build-arg AMG_UTILS_VERSION=$version"
    
    # Tags for local build
    local tags="-t $image_name:$version"
    if [ "$version" != "dev" ]; then
        tags="$tags -t $image_name:local-$version"
    fi
    
    print_info "Building Docker image..."
    docker build \
        $build_args \
        $tags \
        "$docker_context"
    
    print_success "Successfully built Docker image:"
    print_success "  - $image_name:$version"
    if [ "$version" != "dev" ]; then
        print_success "  - $image_name:local-$version"
    fi
    
    print_info "To run the image:"
    print_info "  docker run -it --rm $image_name:$version"
    
    print_info "To test with GPU support:"
    print_info "  docker run -it --rm --gpus all --runtime nvidia $image_name:$version"
}

# Main execution
main() {
    # Parse arguments
    if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
        show_usage
        exit 0
    fi
    
    print_info "AMG Docker Image Local Build Script"
    print_info "=================================="
    
    # Pre-flight checks
    check_docker
    
    # Get version
    local version=$(get_version "$1")
    print_info "Using version: $version"
    
    # Change to repository root if not already there
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local repo_root="$(cd "$script_dir/../.." && pwd)"
    cd "$repo_root"
    
    print_info "Working directory: $(pwd)"
    
    # Execute build
    build_local "$version"
    
    print_success "Local Docker image build completed successfully!"
}

# Run main function with all arguments
main "$@"
