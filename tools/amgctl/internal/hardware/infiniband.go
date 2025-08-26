package hardware

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/prometheus/procfs/sysfs"
)

// getInfinibandDeviceFlags discovers InfiniBand devices on the host and generates
// Docker device flags for essential devices only (rdma_cm and uverbs devices).
// Returns a string containing the Docker device flags.
func GetInfinibandDeviceFlags() (string, error) {
	var deviceFlags []string

	// Always add the rdma_cm device if it exists
	rdmaCmDevice := "/dev/infiniband/rdma_cm"
	if deviceExists(rdmaCmDevice) {
		deviceFlags = append(deviceFlags, fmt.Sprintf("--device=%s", rdmaCmDevice))
	}

	// Discover InfiniBand devices using sysfs to get uverbs devices
	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		return "", fmt.Errorf("failed to initialize sysfs: %v", err)
	}

	ibClass, err := fs.InfiniBandClass()
	if err != nil {
		return "", fmt.Errorf("failed to read InfiniBand class information: %v", err)
	}

	// Generate device flags for uverbs devices only
	for deviceName := range ibClass {
		// Add the main device (uverbs)
		uverbsDevice := fmt.Sprintf("/dev/infiniband/uverbs%s", strings.TrimPrefix(deviceName, "mlx"))
		if deviceExists(uverbsDevice) {
			deviceFlags = append(deviceFlags, fmt.Sprintf("--device=%s", uverbsDevice))
		}
	}

	// Also scan for any uverbs devices that might not be detected via sysfs
	if deviceDirs, err := os.ReadDir("/dev/infiniband"); err == nil {
		for _, entry := range deviceDirs {
			deviceName := entry.Name()
			// Only include uverbs devices and rdma_cm
			if strings.HasPrefix(deviceName, "uverbs") || deviceName == "rdma_cm" {
				devicePath := fmt.Sprintf("/dev/infiniband/%s", deviceName)
				if !containsDevice(deviceFlags, devicePath) {
					deviceFlags = append(deviceFlags, fmt.Sprintf("--device=%s", devicePath))
				}
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

// InfiniBandNetworkInterface represents an InfiniBand network interface with its status
type InfiniBandNetworkInterface struct {
	Name      string
	IPAddress string
	Status    string
}

// GetInfiniBandNetworkInterfaces returns InfiniBand network interfaces with their IP addresses and status
func GetInfiniBandNetworkInterfaces() ([]InfiniBandNetworkInterface, error) {
	var ibInterfaces []InfiniBandNetworkInterface

	// Get all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %v", err)
	}

	// Filter for InfiniBand interfaces (typically named ib0, ib1, etc.)
	for _, iface := range interfaces {
		if strings.HasPrefix(iface.Name, "ib") {
			// Get IP addresses for this interface
			addrs, err := iface.Addrs()
			if err != nil {
				continue // Skip this interface if we can't get addresses
			}

			// Determine interface status
			status := "down"
			if iface.Flags&net.FlagUp != 0 {
				status = "up"
			}

			// If no IP addresses, still show the interface but without IP
			if len(addrs) == 0 {
				ibInterfaces = append(ibInterfaces, InfiniBandNetworkInterface{
					Name:      iface.Name,
					IPAddress: "no IP assigned",
					Status:    status,
				})
				continue
			}

			// Add interface for each IP address
			for _, addr := range addrs {
				// Parse the IP address from CIDR notation
				ip, _, err := net.ParseCIDR(addr.String())
				if err != nil {
					// If not CIDR, try to parse as IP directly
					ip = net.ParseIP(addr.String())
					if ip == nil {
						continue
					}
				}

				// Only include IPv4 and IPv6 addresses (skip link-local etc)
				if ip.To4() != nil || (ip.To16() != nil && !ip.IsLinkLocalUnicast()) {
					ibInterfaces = append(ibInterfaces, InfiniBandNetworkInterface{
						Name:      iface.Name,
						IPAddress: addr.String(),
						Status:    status,
					})
				}
			}
		}
	}

	return ibInterfaces, nil
}
