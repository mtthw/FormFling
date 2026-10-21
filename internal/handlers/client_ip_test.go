package handlers

import (
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestClientIPTrustsOnlyConfiguredProxies(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
		want       string
	}{
		{"direct client cannot spoof", "198.51.100.5:4321", "203.0.113.9", "198.51.100.5"},
		{"trusted proxy", "192.0.2.5:4321", "203.0.113.9", "203.0.113.9"},
		{"use nearest untrusted hop", "192.0.2.5:4321", "198.51.100.8, 203.0.113.9, 192.0.2.6", "203.0.113.9"},
		{"malformed forwarding header", "192.0.2.5:4321", "invalid", "192.0.2.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/submit", nil)
			req.RemoteAddr = tt.remoteAddr
			req.Header.Set("X-Forwarded-For", tt.forwarded)
			if got := clientIP(req, trusted); got != tt.want {
				t.Fatalf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
