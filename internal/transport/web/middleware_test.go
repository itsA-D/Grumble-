package web

import (
	"context"
	"net"
	"net/http/httptest"
	"testing"
)

// TestResolveClientAddrTrustsHeaderOnlyFromTrustedPeer proves the client address
// (and therefore the salted source hash behind the submit rate limit) cannot be
// spoofed by a header from an untrusted peer (NFR-PRIV-1, FR-MOD-2).
func TestResolveClientAddrTrustsHeaderOnlyFromTrustedPeer(t *testing.T) {
	_, trustedNet, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatalf("parse cidr: %v", err)
	}
	trusted := []*net.IPNet{trustedNet}

	tests := []struct {
		name     string
		peer     string
		header   string
		wantAddr string
	}{
		{name: "untrusted peer, spoofed header ignored", peer: "203.0.113.7:5", header: "1.2.3.4", wantAddr: "203.0.113.7"},
		{name: "untrusted peer, no header", peer: "203.0.113.7:5", header: "", wantAddr: "203.0.113.7"},
		{name: "trusted peer, header honoured", peer: "10.1.2.3:5", header: "198.51.100.9", wantAddr: "198.51.100.9"},
		{name: "trusted peer, first hop of list", peer: "10.1.2.3:5", header: "198.51.100.9, 10.0.0.1", wantAddr: "198.51.100.9"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
			r.RemoteAddr = tc.peer
			if tc.header != "" {
				r.Header.Set("X-Forwarded-For", tc.header)
			}
			got := resolveClientAddr(r, "X-Forwarded-For", trusted)
			if got != tc.wantAddr {
				t.Fatalf("resolveClientAddr = %q, want %q", got, tc.wantAddr)
			}
		})
	}
}
