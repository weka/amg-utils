# NicClusterPolicy Configuration Examples

This directory contains example configurations for the NicClusterPolicy feature in the amg-chart.

## H100A Hardware Configuration

The `h100a-values.yaml` file shows how to configure the NicClusterPolicy for H100A hardware.

To deploy with H100A configuration:

```bash
helm install my-release amg-chart/ -f examples/h100a-values.yaml
```

## Custom Hardware Configuration

For other hardware configurations, you'll need to:

1. Determine the correct interface names for your hardware
2. Update the `nicClusterPolicy.rdmaSharedDevicePlugin.selectors.ifNames` list
3. Optionally adjust other OFED driver or RDMA plugin settings

Example for custom hardware:

```yaml
nicClusterPolicy:
  enabled: true
  name: "my-custom-nic-cluster-policy"
  rdmaSharedDevicePlugin:
    selectors:
      ifNames:
        - "ib0"
        - "ib1"
        # Add your specific interface names here
```

## Finding Interface Names

To find the correct interface names for your hardware, you can run:

```bash
# On nodes with RDMA hardware
ip link show | grep ib
# or
ls /sys/class/net/ | grep ib
```

## Disabling NicClusterPolicy

By default, the NicClusterPolicy is disabled. To keep it disabled:

```yaml
nicClusterPolicy:
  enabled: false
```
