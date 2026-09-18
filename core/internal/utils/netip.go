package utils

import (
	"net"
	"net/http"
	"strings"
)

// ParseHostNoPort returns the host part (no port) from strings like "ip:port", "[v6]:port", or "ip".
func ParseHostNoPort(s string) string {
	if s == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(s); err == nil {
		return h
	}
	return s
}

// FirstForwardedFor returns the first IP from X-Forwarded-For (left-most), trimmed.
func FirstForwardedFor(xff string) string {
	xff = strings.TrimSpace(xff)
	if xff == "" {
		return ""
	}
	if i := strings.IndexByte(xff, ','); i >= 0 {
		xff = xff[:i]
	}
	return strings.TrimSpace(xff)
}

// ForwardedFor returns the first valid for= value from RFC 7239 Forwarded.
func ForwardedFor(value string) string {
	for _, element := range strings.Split(value, ",") {
		for _, pair := range strings.Split(element, ";") {
			key, raw, ok := strings.Cut(strings.TrimSpace(pair), "=")
			if !ok || !strings.EqualFold(key, "for") {
				continue
			}
			if ip := parseHeaderIP(raw); ip != "" {
				return ip
			}
		}
	}
	return ""
}

// ClientIP resolves the client IP only when the direct network peer is an
// explicitly trusted proxy. Otherwise, forwarding headers are ignored.
func ClientIP(r *http.Request, trustProxy bool, trustedProxies *IPMatcher) string {
	remoteIP := ParseHostNoPort(r.RemoteAddr)
	if !trustProxy || trustedProxies == nil || !trustedProxies.Allow(remoteIP) {
		return remoteIP
	}
	if ip := ForwardedFor(r.Header.Get("Forwarded")); ip != "" {
		return ip
	}
	if ip := parseHeaderIP(FirstForwardedFor(r.Header.Get("X-Forwarded-For"))); ip != "" {
		return ip
	}
	if ip := parseHeaderIP(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if ip := parseHeaderIP(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	return remoteIP
}

func parseHeaderIP(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "[") {
		if end := strings.IndexByte(value, ']'); end > 1 {
			value = value[1:end]
		}
	}
	ip := ParseHostNoPort(value)
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}

// IPMatcher matches exact IPs and CIDRs.
type IPMatcher struct {
	ips  []net.IP
	nets []*net.IPNet
}

func NewIPMatcher(list []string) *IPMatcher {
	m := &IPMatcher{}
	for _, raw := range list {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if _, ipnet, err := net.ParseCIDR(s); err == nil {
			m.nets = append(m.nets, ipnet)
			continue
		}
		if ip := net.ParseIP(s); ip != nil {
			m.ips = append(m.ips, ip)
		}
	}
	return m
}

func (m *IPMatcher) IsEmpty() bool {
	return len(m.ips) == 0 && len(m.nets) == 0
}

func (m *IPMatcher) Allow(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, v := range m.ips {
		if v.Equal(ip) {
			return true
		}
	}
	for _, n := range m.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
