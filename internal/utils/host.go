package utils

import (
	"net"
	"os"
)

// GetHostname returns the hostname of the current machine.
// It first checks environment variables HOSTNAME and HOST_NAME,
// then falls back to os.Hostname().
// Returns "unknown" if hostname cannot be determined.
func GetHostname() string {
	// Try to get hostname from environment variable first
	if hostname := os.Getenv("HOSTNAME"); hostname != "" {
		return hostname
	}
	if hostname := os.Getenv("HOST_NAME"); hostname != "" {
		return hostname
	}

	// Get hostname from system
	hostname, err := os.Hostname()
	if err != nil {
		// Fallback to a default value if hostname cannot be determined
		return "unknown"
	}
	return hostname
}

// GetHostIP returns the primary IP address of the current machine.
// It first checks the HOST_IP environment variable,
// then attempts to get the IP by connecting to a dummy address (8.8.8.8:80),
// and finally falls back to resolving the hostname.
// Returns "unknown" if IP cannot be determined.
func GetHostIP() string {
	// Try to get IP from environment variable first
	if ip := os.Getenv("HOST_IP"); ip != "" {
		return ip
	}

	// Get primary IP address by connecting to a dummy address
	// This method gets the IP that would be used for outbound connections
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// Fallback: try to get IP from hostname
		hostname, err := os.Hostname()
		if err != nil {
			return "unknown"
		}
		addrs, err := net.LookupIP(hostname)
		if err != nil || len(addrs) == 0 {
			return "unknown"
		}
		// Return first IPv4 address
		for _, addr := range addrs {
			if ipv4 := addr.To4(); ipv4 != nil {
				return ipv4.String()
			}
		}
		return "unknown"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
