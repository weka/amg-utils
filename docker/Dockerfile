# Use NVIDIA CUDA 12.9.1 development image for Ubuntu 24.04
FROM nvidia/cuda:12.9.1-devel-ubuntu24.04

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1
# TODO: Support more architectures besides Hopper
# TODO: Do we actually need to set that anymore? Check!
ENV TORCH_CUDA_ARCH_LIST="9.0"
ENV FORCE_CUDA="1"
ENV CUDA_HOME=/usr/local/cuda
ENV NVIDIA_GDS=enabled

# amg-utils version
ARG AMG_UTILS_VERSION=v0.1.7

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

# Setup host with amgctl (use upstream LMCache and VLLM)
RUN amgctl host setup --lmcache-repo "https://github.com/LMCache/LMCache.git" --lmcache-commit "v0.3.3" --vllm-version "0.10.0" --skip-hotfixes

# Activate uv environment on login
RUN echo "source /root/amg_stable/.venv/bin/activate" >> /root/.bashrc

# Set working directory
WORKDIR /root/amg_stable

# Set default command
CMD ["/bin/bash"] 