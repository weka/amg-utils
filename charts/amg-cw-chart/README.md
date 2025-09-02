# AMG CW Chart

This Helm chart deploys two DaemonSets designed for CW AMG workloads:

1. **install-nvidiafs**: Installs NVIDIA GDS (GPU Direct Storage) and loads the nvidia_fs kernel module
2. **weka-amg**: Deploys the AMG container with RDMA and GPU resources for Weka filesystem integration

## Prerequisites

- Kubernetes cluster with nodes that have NVIDIA GPUs
- RDMA/InfiniBand networking support
- Persistent Volume Claim named `wekafs-amg` (or configure a different name in values)

## Installation

```bash
# Install with default values
helm install my-amg-cw ./charts/amg-cw-chart

# Install with custom values
helm install my-amg-cw ./charts/amg-cw-chart -f my-values.yaml
```

## Configuration

The following table lists the main configurable parameters:

### NVIDIA FS DaemonSet

| Parameter | Description | Default |
|-----------|-------------|---------|
| `nvidiaFs.enabled` | Enable NVIDIA FS DaemonSet | `true` |
| `nvidiaFs.image.repository` | Ubuntu image repository | `ubuntu` |
| `nvidiaFs.image.tag` | Ubuntu image tag | `22.04` |
| `nvidiaFs.kernelVersion` | Kernel version for GDS installation | `6.5.13-65-650-4141-22041-coreweave-amd64-85c45edc` |
| `nvidiaFs.nvidiaGdsVersion` | NVIDIA GDS package version | `12.9.1-1` |
| `nvidiaFs.resources` | Resource requests and limits | See values.yaml |

### Weka AMG DaemonSet

| Parameter | Description | Default |
|-----------|-------------|---------|
| `wekaAmg.enabled` | Enable Weka AMG DaemonSet | `true` |
| `wekaAmg.image.repository` | AMG image repository | `sdimitro509/amg` |
| `wekaAmg.image.tag` | AMG image tag | `latest` |
| `wekaAmg.resources.requests.nvidiaGpu` | Number of GPUs to request | `8` |
| `wekaAmg.resources.requests.rdmaIb` | Number of RDMA IB devices | `1` |
| `wekaAmg.pvc.name` | PVC name for Weka filesystem | `wekafs-amg` |
| `wekaAmg.env.*` | NCCL and UCX environment variables | See values.yaml |

## Example Values

```yaml
# Disable NVIDIA FS installation if already done
nvidiaFs:
  enabled: false

# Customize AMG resources for smaller nodes
wekaAmg:
  resources:
    requests:
      nvidiaGpu: 4
      cpu: "8"
      memory: "64Gi"
    limits:
      nvidiaGpu: 4
      cpu: "8"
      memory: "64Gi"
  
  # Use different PVC
  pvc:
    name: "my-weka-pvc"
```

## Uninstallation

```bash
helm uninstall my-amg-cw
```

Note: The NVIDIA FS kernel module will remain loaded on the nodes even after uninstallation.
