package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPUsesForwardedHeadersFromTrustedProxy(t *testing.T) {
	r := httptest.NewRequest("GET", "http://navire.test/", http.NoBody)
	r.RemoteAddr = "172.19.0.15:43120"
	r.Header.Set("Forwarded", `for="[2001:db8::10]:443";proto=https`)
	r.Header.Set("X-Forwarded-For", "198.51.100.20")

	got := ClientIP(r, true, NewIPMatcher([]string{"172.19.0.0/16"}))
	if got != "2001:db8::10" {
		t.Fatalf("ClientIP() = %q, want 2001:db8::10", got)
	}
}

func TestClientIPUsesXForwardedForFallback(t *testing.T) {
	r := httptest.NewRequest("GET", "http://navire.test/", http.NoBody)
	r.RemoteAddr = "172.19.0.15:43120"
	r.Header.Set("X-Forwarded-For", "198.51.100.20, 172.19.0.15")

	got := ClientIP(r, true, NewIPMatcher([]string{"172.19.0.0/16"}))
	if got != "198.51.100.20" {
		t.Fatalf("ClientIP() = %q, want 198.51.100.20", got)
	}
}

func TestClientIPIgnoresForwardedHeadersFromUntrustedPeer(t *testing.T) {
	r := httptest.NewRequest("GET", "http://navire.test/", http.NoBody)
	r.RemoteAddr = "10.70.80.2:43120"
	r.Header.Set("X-Forwarded-For", "198.51.100.20")
	r.Header.Set("X-Real-IP", "198.51.100.20")

	got := ClientIP(r, true, NewIPMatcher([]string{"172.19.0.0/16"}))
	if got != "10.70.80.2" {
		t.Fatalf("ClientIP() = %q, want 10.70.80.2", got)
	}
}

func TestClientIPIgnoresForwardedHeadersWhenProxyTrustIsDisabled(t *testing.T) {
	r := httptest.NewRequest("GET", "http://navire.test/", http.NoBody)
	r.RemoteAddr = "172.19.0.15:43120"
	r.Header.Set("X-Forwarded-For", "198.51.100.20")

	got := ClientIP(r, false, NewIPMatcher([]string{"172.19.0.0/16"}))
	if got != "172.19.0.15" {
		t.Fatalf("ClientIP() = %q, want 172.19.0.15", got)
	}
}
