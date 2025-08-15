package hardware

import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

// getGpuCount returns the number of available NVIDIA GPUs on the host system.
// It initializes the NVML library, queries the device count, and properly shuts down the library.
// Returns the GPU count and any error encountered during the process.
func GetGpuCount() (int, error) {
	// Initialize the NVML library
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return 0, fmt.Errorf("failed to initialize NVML: %v", nvml.ErrorString(ret))
	}

	// Ensure proper cleanup of NVML resources
	defer func() {
		if shutdownRet := nvml.Shutdown(); shutdownRet != nvml.SUCCESS {
			// Log shutdown error, but don't override the main error
			fmt.Printf("Warning: failed to shutdown NVML: %v\n", nvml.ErrorString(shutdownRet))
		}
	}()

	// Get the number of NVIDIA devices
	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return 0, fmt.Errorf("failed to get device count: %v", nvml.ErrorString(ret))
	}

	return count, nil
}

// GetGpuInfo returns basic information about all available NVIDIA GPUs.
// This is a helper function that can be used for more detailed hardware discovery.
func GetGpuInfo() ([]string, error) {
	count, err := GetGpuCount()
	if err != nil {
		return nil, err
	}

	if count == 0 {
		return []string{}, nil
	}

	// Initialize NVML again for device info queries
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to initialize NVML: %v", nvml.ErrorString(ret))
	}
	defer func() {
		if shutdownRet := nvml.Shutdown(); shutdownRet != nvml.SUCCESS {
			fmt.Printf("Warning: failed to shutdown NVML in GetGpuInfo: %v\n", nvml.ErrorString(shutdownRet))
		}
	}()

	var gpuInfo []string
	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get device handle for GPU %d: %v", i, nvml.ErrorString(ret))
		}

		name, ret := device.GetName()
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("failed to get device name for GPU %d: %v", i, nvml.ErrorString(ret))
		}

		gpuInfo = append(gpuInfo, fmt.Sprintf("GPU %d: %s", i, name))
	}

	return gpuInfo, nil
}
