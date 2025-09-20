package utils

import (
	"testing"
	"fmt"
)

//test UrlParse function
func TestParseURL(t *testing.T) {
	tests := []struct {
		isHttps  bool
		url      string
		hostname string
		port     int
		subpath  string
	}{
		{false, "http://example.com/api/echo", "example.com", 80, "/api/echo"},
		{true, "https://example.com:8080/api/echo", "example.com", 8080, "/api/echo"},
		{false, "http://example.com", "example.com", 80, "/"},
		{true, "https://example.com:443", "example.com", 443, "/"},
		{true, "https://example.com/resource", "example.com", 443, "/resource"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			isHttps, hostname, port, subpath, err := ParseURL(tt.url)
			fmt.Printf("test result, isHttps: %v, hostname: %s, port: %d, subpath: %s, err: %v\n", isHttps, hostname, port, subpath, err)
			if isHttps != tt.isHttps {
				t.Errorf("ParseURL(%q) = %v, want %v", tt.url, isHttps, tt.isHttps)
			}
			if err != nil {
				t.Errorf("ParseURL(%q) returned error: %v", tt.url, err)
				return
			}
			if hostname != tt.hostname {
				t.Errorf("ParseURL(%q) = %q, want %q", tt.url, hostname, tt.hostname)
			}
			if port != tt.port {
				t.Errorf("ParseURL(%q) = %d, want %d", tt.url, port, tt.port)
			}
			if subpath != tt.subpath {
				t.Errorf("ParseURL(%q) = %q, want %q", tt.url, subpath, tt.subpath)
			}
		})
	}
}
