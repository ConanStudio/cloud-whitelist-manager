package ip

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/ConanStudio/cloud-whitelist-manager/internal/config"
)

// GetPublicIP retrieves public IP using the configured IP sources
func GetPublicIP(sources []config.IPSource) (string, error) {
	for _, source := range sources {
		ip, err := getIPFromSource(source)
		if err == nil && ip != "" {
			return ip, nil
		}
		// If this source fails, try the next one
	}
	return "", fmt.Errorf("failed to get IP from all configured sources")
}

// getIPFromSource retrieves IP from a specific source
func getIPFromSource(source config.IPSource) (string, error) {
	switch source.Type {
	case "http":
		return getIPFromHTTP(source)
	case "command":
		return getIPFromCommand(source)
	case "interface":
		return getIPFromInterface(source)
	default:
		return "", fmt.Errorf("unknown IP source type: %s", source.Type)
	}
}

// getIPFromHTTP retrieves IP from HTTP endpoint
func getIPFromHTTP(source config.IPSource) (string, error) {
	client := &http.Client{
		Timeout: time.Duration(source.Timeout) * time.Second,
	}

	req, err := http.NewRequest("GET", source.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Add custom headers
	for key, value := range source.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	if err != nil && n == 0 {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	ip := strings.TrimSpace(string(buf[:n]))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	return ip, nil
}

// getIPFromCommand retrieves IP by executing a command
func getIPFromCommand(source config.IPSource) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(source.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", source.Cmd)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %v", err)
	}

	ip := strings.TrimSpace(string(output))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	return ip, nil
}

// getIPFromInterface retrieves IP from network interface
func getIPFromInterface(source config.IPSource) (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %v", err)
	}

	for _, iface := range interfaces {
		if iface.Name == source.Interface {
			addrs, err := iface.Addrs()
			if err != nil {
				return "", fmt.Errorf("failed to get addresses for interface %s: %v", source.Interface, err)
			}

			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}

				// Skip loopback addresses
				if ip.IsLoopback() {
					continue
				}

				// Check IPv6 if requested
				if source.IPv6 && ip.To4() == nil {
					return ip.String(), nil
				}

				// Check IPv4 if not specifically requesting IPv6
				if !source.IPv6 && ip.To4() != nil {
					return ip.String(), nil
				}
			}
		}
	}

	return "", fmt.Errorf("interface %s not found or has no valid IP", source.Interface)
}