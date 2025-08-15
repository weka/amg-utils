package hardware

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/prometheus/procfs/sysfs"
)

// getInfinibandDeviceFlags discovers all InfiniBand devices on the host and generates
// Docker device flags for each device and port. It also includes the rdma_cm device.
// Returns a string containing all the Docker device flags.
func GetInfinibandDeviceFlags() (string, error) {
	var deviceFlags []string

	// Discover InfiniBand devices using sysfs
	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		return "", fmt.Errorf("failed to initialize sysfs: %v", err)
	}

	ibClass, err := fs.InfiniBandClass()
	if err != nil {
		return "", fmt.Errorf("failed to read InfiniBand class information: %v", err)
	}

	// Generate device flags for each InfiniBand device and its ports
	for deviceName, device := range ibClass {
		// Add the main device (uverbs)
		uverbsDevice := fmt.Sprintf("/dev/infiniband/uverbs%s", strings.TrimPrefix(deviceName, "mlx"))
		if deviceExists(uverbsDevice) {
			deviceFlags = append(deviceFlags, fmt.Sprintf("--device=%s", uverbsDevice))
		}

		// Add device flags for each port
		for portNum := range device.Ports {
			// Add the port-specific device if it exists
			portDevice := fmt.Sprintf("/dev/infiniband/%s_%d", deviceName, portNum)
			if deviceExists(portDevice) {
				deviceFlags = append(deviceFlags, fmt.Sprintf("--device=%s", portDevice))
			}
		}

		// Add issm device if it exists
		issmDevice := fmt.Sprintf("/dev/infiniband/issm%s", strings.TrimPrefix(deviceName, "mlx"))
		if deviceExists(issmDevice) {
			deviceFlags = append(deviceFlags, fmt.Sprintf("--device=%s", issmDevice))
		}

		// Add umad devices for each port
		for portNum := range device.Ports {
			umadDevice := fmt.Sprintf("/dev/infiniband/umad%s_%d", strings.TrimPrefix(deviceName, "mlx"), portNum)
			if deviceExists(umadDevice) {
				deviceFlags = append(deviceFlags, fmt.Sprintf("--device=%s", umadDevice))
			}
		}
	}

	// Add the rdma_cm device
	rdmaCmDevice := "/dev/infiniband/rdma_cm"
	if deviceExists(rdmaCmDevice) {
		deviceFlags = append(deviceFlags, fmt.Sprintf("--device=%s", rdmaCmDevice))
	}

	// Discover all available device files by scanning the directory
	if deviceDirs, err := os.ReadDir("/dev/infiniband"); err == nil {
		for _, entry := range deviceDirs {
			devicePath := fmt.Sprintf("/dev/infiniband/%s", entry.Name())
			if !containsDevice(deviceFlags, devicePath) {
				deviceFlags = append(deviceFlags, fmt.Sprintf("--device=%s", devicePath))
			}
		}
	}

	return strings.Join(deviceFlags, " "), nil
}

// deviceExists checks if a device file exists
func deviceExists(devicePath string) bool {
	_, err := os.Stat(devicePath)
	return err == nil
}

// containsDevice checks if a device is already in the flags list
func containsDevice(flags []string, device string) bool {
	deviceFlag := fmt.Sprintf("--device=%s", device)
	for _, flag := range flags {
		if flag == deviceFlag {
			return true
		}
	}
	return false
}

// GetInfinibandDeviceInfo returns detailed information about InfiniBand devices
// This is a helper function for debugging and information display
func GetInfinibandDeviceInfo() ([]string, error) {
	var deviceInfo []string

	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sysfs: %v", err)
	}

	ibClass, err := fs.InfiniBandClass()
	if err != nil {
		return nil, fmt.Errorf("failed to read InfiniBand class information: %v", err)
	}

	if len(ibClass) == 0 {
		return []string{"No InfiniBand devices found"}, nil
	}

	for deviceName, device := range ibClass {
		deviceInfo = append(deviceInfo, fmt.Sprintf("InfiniBand Device: %s", deviceName))

		// Add board ID if available
		if device.BoardID != "" {
			deviceInfo = append(deviceInfo, fmt.Sprintf("  Board ID: %s", device.BoardID))
		}

		// Add firmware version if available
		if device.FirmwareVersion != "" {
			deviceInfo = append(deviceInfo, fmt.Sprintf("  Firmware Version: %s", device.FirmwareVersion))
		}

		// List ports
		if len(device.Ports) > 0 {
			deviceInfo = append(deviceInfo, fmt.Sprintf("  Ports: %d", len(device.Ports)))
			for portNum, port := range device.Ports {
				deviceInfo = append(deviceInfo, fmt.Sprintf("    Port %d: State=%s",
					portNum, port.State))
			}
		}
	}

	// Check for actual device files
	deviceInfo = append(deviceInfo, "Available device files:")
	deviceDirs := []string{"/dev/infiniband"}

	for _, dir := range deviceDirs {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				devicePath := filepath.Join(dir, entry.Name())
				deviceInfo = append(deviceInfo, fmt.Sprintf("  %s", devicePath))
			}
		}
	}

	return deviceInfo, nil
}
