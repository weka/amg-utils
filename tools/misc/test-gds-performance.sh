#!/bin/bash

set -e

# Default values
WEKA_PATH="/mnt/weka"
WORKERS=48
SIZE="2G"
BLOCK_SIZE="1m"
CONTAINER_NAME="amg"
DURATION=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

usage() {
    echo "Usage: $0 [OPTIONS] <host|container|both>"
    echo ""
    echo "Test GDS performance on host, container, or both"
    echo ""
    echo "OPTIONS:"
    echo "  -p PATH     WekaFS mount path (default: /mnt/weka)"
    echo "  -w WORKERS  Number of workers (default: 48)"
    echo "  -s SIZE     Test file size (default: 2G)"
    echo "  -b SIZE     Block size (default: 1m)"
    echo "  -c NAME     Container name (default: amg)"
    echo "  -T SECONDS  Duration in seconds for gdsio execution (optional)"
    echo "  -h          Show this help"
    echo ""
    echo "EXAMPLES:"
    echo "  $0 host                    # Test on host only"
    echo "  $0 container               # Test in container only"
    echo "  $0 both                    # Test both and compare"
    echo "  $0 -p /weka -w 32 both     # Custom path and workers"
    echo "  $0 -T 60 both              # Run each test for 60 seconds duration"
}

log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    exit 1
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

run_host_test() {
    local test_type=$1
    local io_flag=$2
    
    log "Running ${test_type} test on HOST..."
    
    export CUFILE_ENV_PATH_JSON="/etc/cufile.json"
    
    /usr/local/cuda/gds/tools/gdsio \
        -D ${WEKA_PATH}/gdsio0 -d 0 -n 0 -w $WORKERS \
        -D ${WEKA_PATH}/gdsio1 -d 1 -n 0 -w $WORKERS \
        -D ${WEKA_PATH}/gdsio2 -d 2 -n 0 -w $WORKERS \
        -D ${WEKA_PATH}/gdsio3 -d 3 -n 0 -w $WORKERS \
        -D ${WEKA_PATH}/gdsio4 -d 4 -n 1 -w $WORKERS \
        -D ${WEKA_PATH}/gdsio5 -d 5 -n 1 -w $WORKERS \
        -D ${WEKA_PATH}/gdsio6 -d 6 -n 1 -w $WORKERS \
        -D ${WEKA_PATH}/gdsio7 -d 7 -n 1 -w $WORKERS \
        -s $SIZE -i $BLOCK_SIZE -x 0 -I $io_flag \
        ${DURATION:+-T $DURATION}
}

run_container_test() {
    local test_type=$1
    local io_flag=$2
    
    log "Running ${test_type} test in CONTAINER..."
    
    docker run --rm \
        --gpus all --runtime=nvidia \
        --device /dev/infiniband/rdma_cm \
        $(for d in /dev/infiniband/uverbs*; do printf -- '--device=%s ' "$d"; done) \
        -v ${WEKA_PATH}:${WEKA_PATH} \
        --env NVIDIA_GDS=enabled \
        --env CUFILE_ENV_PATH_JSON="/etc/cufile.json" \
        -v /etc/cufile.json:/etc/cufile.json:ro \
        --cap-add=IPC_LOCK \
        --network=host \
        $CONTAINER_NAME \
        gdsio \
            -D ${WEKA_PATH}/gdsio0 -d 0 -n 0 -w $WORKERS \
            -D ${WEKA_PATH}/gdsio1 -d 1 -n 0 -w $WORKERS \
            -D ${WEKA_PATH}/gdsio2 -d 2 -n 0 -w $WORKERS \
            -D ${WEKA_PATH}/gdsio3 -d 3 -n 0 -w $WORKERS \
            -D ${WEKA_PATH}/gdsio4 -d 4 -n 1 -w $WORKERS \
            -D ${WEKA_PATH}/gdsio5 -d 5 -n 1 -w $WORKERS \
            -D ${WEKA_PATH}/gdsio6 -d 6 -n 1 -w $WORKERS \
            -D ${WEKA_PATH}/gdsio7 -d 7 -n 1 -w $WORKERS \
            -s $SIZE -i $BLOCK_SIZE -x 0 -I $io_flag \
            ${DURATION:+-T $DURATION}
}

cleanup_files() {
    log "Cleaning up test files..."
    for i in {0..7}; do
        if [ -f "${WEKA_PATH}/gdsio${i}" ]; then
            rm -f "${WEKA_PATH}/gdsio${i}"
        fi
    done
}

# Parse command line arguments
while getopts "p:w:s:b:c:T:h" opt; do
    case $opt in
        p) WEKA_PATH="$OPTARG" ;;
        w) WORKERS="$OPTARG" ;;
        s) SIZE="$OPTARG" ;;
        b) BLOCK_SIZE="$OPTARG" ;;
        c) CONTAINER_NAME="$OPTARG" ;;
        T) DURATION="$OPTARG" ;;
        h) usage; exit 0 ;;
        *) usage; exit 1 ;;
    esac
done

shift $((OPTIND-1))

if [ $# -ne 1 ]; then
    error "Please specify test target: host, container, or both"
fi

TARGET="$1"

# Validate target
if [[ "$TARGET" != "host" && "$TARGET" != "container" && "$TARGET" != "both" ]]; then
    error "Invalid target. Use: host, container, or both"
fi

# Check if WekaFS path exists
if [ ! -d "$WEKA_PATH" ]; then
    error "WekaFS path does not exist: $WEKA_PATH"
fi

# Display test configuration
echo -e "${YELLOW}GDS Performance Test Configuration:${NC}"
echo "  WekaFS Path: $WEKA_PATH"
echo "  Workers: $WORKERS"
echo "  File Size: $SIZE"
echo "  Block Size: $BLOCK_SIZE"
echo "  Container: $CONTAINER_NAME"
echo "  Target: $TARGET"
if [ -n "$DURATION" ]; then
    echo "  Duration: ${DURATION}s"
fi
echo ""

# Cleanup any existing test files
cleanup_files

# Run tests based on target
case $TARGET in
    "host")
        echo -e "${GREEN}=== HOST PERFORMANCE TEST ===${NC}"
        run_host_test "WRITE" 1
        echo ""
        run_host_test "READ" 0
        ;;
    "container")
        echo -e "${GREEN}=== CONTAINER PERFORMANCE TEST ===${NC}"
        run_container_test "WRITE" 1
        echo ""
        run_container_test "READ" 0
        ;;
    "both")
        echo -e "${GREEN}=== HOST vs CONTAINER PERFORMANCE COMPARISON ===${NC}"
        
        echo -e "\n${BLUE}--- HOST WRITE TEST ---${NC}"
        run_host_test "WRITE" 1
        
        echo -e "\n${BLUE}--- CONTAINER WRITE TEST ---${NC}"
        run_container_test "WRITE" 1
        
        echo -e "\n${BLUE}--- HOST READ TEST ---${NC}"
        run_host_test "READ" 0
        
        echo -e "\n${BLUE}--- CONTAINER READ TEST ---${NC}"
        run_container_test "READ" 0
        
        echo ""
        success "Performance comparison complete!"
        warn "Compare the throughput numbers above - they should be nearly identical."
        ;;
esac

# Final cleanup
cleanup_files
success "Test completed successfully!"
