# Use NVIDIA CUDA 12.9.1 development image for Ubuntu 24.04
FROM nvidia/cuda:12.9.1-devel-ubuntu24.04

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1
# Add support for new architectures here
ENV TORCH_CUDA_ARCH_LIST="9.0 10.0+PTX"
ENV FORCE_CUDA="1"
ENV CUDA_HOME=/usr/local/cuda
ENV NVIDIA_GDS=enabled

# amg-utils version
ARG AMG_UTILS_VERSION=v0.1.16

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
    python3-venv \
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

# Set working directory
WORKDIR /root/amg_stable

# Activate uv environment on login
RUN uv venv .venv
RUN echo "source /root/amg_stable/.venv/bin/activate" >> /root/.bashrc

COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh

# Run AMGCTL at container start time where nvidia runtime is available - supports cross and local builds
ENTRYPOINT [ "entrypoint.sh" ]
# Set default command
CMD ["/bin/bash"] 
