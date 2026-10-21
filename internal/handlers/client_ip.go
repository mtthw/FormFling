package handlers

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

func clientIP(r *http.Request, trustedProxies []netip.Prefix) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return r.RemoteAddr
	}
	peer = peer.Unmap()
	if !isTrustedProxy(peer, trustedProxies) {
		return peer.String()
	}

	// Walk from the nearest proxy toward the visitor. Ignore any values a
	// visitor could have supplied before the first untrusted address.
	forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(forwarded) - 1; i >= 0; i-- {
		addr, err := netip.ParseAddr(strings.TrimSpace(forwarded[i]))
		if err != nil {
			return peer.String()
		}
		addr = addr.Unmap()
		if !isTrustedProxy(addr, trustedProxies) {
			return addr.String()
		}
	}
	return peer.String()
}

func isTrustedProxy(addr netip.Addr, trustedProxies []netip.Prefix) bool {
	for _, prefix := range trustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
