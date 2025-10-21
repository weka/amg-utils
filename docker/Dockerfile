# ==============================================================================
# Multi-stage Dockerfile for AMG Utils
# Creates two image variants:
#   - amg-vanilla: Base image with amgctl installed
#   - amg-poc: Full image with amgctl host setup and environment configured
# ==============================================================================

# ==============================================================================
# Base stage: Common dependencies and amgctl installation
# ==============================================================================
FROM nvidia/cuda:12.9.1-devel-ubuntu24.04 AS base

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1
# Add support for new architectures here
ENV TORCH_CUDA_ARCH_LIST="8.0 9.0 10.0+PTX"
ENV FORCE_CUDA="1"
ENV CUDA_HOME=/usr/local/cuda
ENV NVIDIA_GDS=enabled

# amg-utils version
ARG AMG_UTILS_VERSION=v0.1.21

# Convenience variables
ENV PATH=/usr/local/cuda/gds/tools:${PATH}
ENV LD_LIBRARY_PATH=/usr/local/cuda/targets/x86_64-linux/lib:${LD_LIBRARY_PATH}

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    wget \
    curl \
    git \
    vim \
    tmux \
    bash \
    htop \
    mtr \
    build-essential \
    cmake \
    pkg-config \
    libssl-dev \
    libffi-dev \
    python3-dev \
    gds-tools-12-9 \
    libcufile-12-9 \
    librdmacm1 \
    ibverbs-providers \
    iproute2 \
    ibutils \
    rdma-core \
    less \
    kmod \
    libibverbs1 \
    libibverbs-dev \
    librdmacm-dev \
    libcufile-rdma1 \
    infiniband-diags \
    ibverbs-utils \
    && rm -rf /var/lib/apt/lists/*

# Clone the repository
RUN git clone --branch ${AMG_UTILS_VERSION} https://github.com/weka/amg-utils.git /root/amg-utils

# Set bash as the default shell
SHELL ["/bin/bash", "-c"]

# Install uv
RUN curl -LsSf https://astral.sh/uv/install.sh | sh

# Add uv to the PATH
ENV PATH="/root/.local/bin:${PATH}"

# Download and install amgctl
RUN wget https://github.com/weka/amg-utils/releases/download/${AMG_UTILS_VERSION}/amgctl-linux-amd64 -O /usr/local/bin/amgctl && \
    chmod +x /usr/local/bin/amgctl

# Create entrypoint script for container initialization
RUN echo '#!/bin/bash\n\
set -e\n\
\n\
# Create amg_stable directory if it does not exist\n\
mkdir -p /root/amg_stable\n\
\n\
# Run amgctl host config cufile on first boot\n\
amgctl host config cufile\n\
\n\
# Execute the main command\n\
exec "$@"' > /usr/local/bin/docker-entrypoint.sh && \
    chmod +x /usr/local/bin/docker-entrypoint.sh

# ==============================================================================
# Vanilla variant: Just the base image with amgctl installed
# ==============================================================================
FROM base AS vanilla

# Set working directory
WORKDIR /root

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

# Set default command
CMD ["/bin/bash"]

# ==============================================================================
# POC variant: Full setup with amgctl host setup and environment configured
# ==============================================================================
FROM base AS poc

# Setup host with amgctl (use upstream LMCache and VLLM)
# The amgctl binary now automatically detects CUDA version and uses appropriate PyTorch index
RUN amgctl host setup

# Activate uv environment on login
RUN echo "source /root/amg_stable/.venv/bin/activate" >> /root/.bashrc

# Set working directory
WORKDIR /root/amg_stable

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

# Set default command
CMD ["/bin/bash"] 