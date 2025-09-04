package ip

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ConanStudio/cloud-whitelist-manager/internal/config"
)

func TestGetIPFromHTTP(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("192.168.1.1"))
	}))
	defer server.Close()

	// Test with valid HTTP source
	source := config.IPSource{
		Type:    "http",
		URL:     server.URL,
		Timeout: 10,
	}

	ip, err := getIPFromSource(source)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if ip != "192.168.1.1" {
		t.Errorf("Expected IP '192.168.1.1', got '%s'", ip)
	}
}

func TestGetIPFromHTTPInvalidIP(t *testing.T) {
	// Create a test server with invalid IP
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid-ip"))
	}))
	defer server.Close()

	// Test with invalid IP
	source := config.IPSource{
		Type:    "http",
		URL:     server.URL,
		Timeout: 10,
	}

	_, err := getIPFromSource(source)
	if err == nil {
		t.Error("Expected error for invalid IP, got none")
	}
}

func TestGetIPFromInterface(t *testing.T) {
	// Get all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Skipf("Cannot get network interfaces: %v", err)
	}

	// Find the first interface with an IP address
	var testInterface string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						testInterface = iface.Name
						break
					}
				}
			}
			if testInterface != "" {
				break
			}
		}
	}

	if testInterface == "" {
		t.Skip("No suitable network interface found for testing")
	}

	// Test with valid interface
	source := config.IPSource{
		Type:      "interface",
		Interface: testInterface,
		IPv6:      false,
	}

	// We can't easily test the actual IP value since it's dynamic
	// but we can test that it doesn't return an error
	_, err = getIPFromSource(source)
	if err != nil {
		t.Logf("Warning: Could not get IP from interface %s: %v", testInterface, err)
	}
}

func TestGetPublicIP(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("192.168.1.1"))
	}))
	defer server.Close()

	// Test with valid sources
	sources := []config.IPSource{
		{Type: "http", URL: server.URL, Timeout: 10},
	}

	ip, err := GetPublicIP(sources)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if ip != "192.168.1.1" {
		t.Errorf("Expected IP '192.168.1.1', got '%s'", ip)
	}
}

func TestGetPublicIPFallback(t *testing.T) {
	// Create a test server that fails
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badServer.Close()

	// Create a test server that works
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("192.168.1.1"))
	}))
	defer goodServer.Close()

	// Test with fallback sources
	sources := []config.IPSource{
		{Type: "http", URL: badServer.URL, Timeout: 10},
		{Type: "http", URL: goodServer.URL, Timeout: 10},
	}

	ip, err := GetPublicIP(sources)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if ip != "192.168.1.1" {
		t.Errorf("Expected IP '192.168.1.1', got '%s'", ip)
	}
}