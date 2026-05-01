// Package fetcher provides HTTP fetching with SSRF protection and safe redirects.
package fetcher

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/user/paimon-mcp-fetch/internal/domain"
)

// ssrfGuard validates URLs against SSRF policies.
type ssrfGuard struct {
	policy domain.SSRFPolicy
}

// newSSRFGuard creates a new SSRF guard with the given policy.
func newSSRFGuard(policy domain.SSRFPolicy) *ssrfGuard {
	return &ssrfGuard{policy: policy}
}

// validate checks if a URL is safe to fetch.
// Returns the parsed URL if safe, or an error describing why it's blocked.
func (g *ssrfGuard) validate(rawURL string) (*url.URL, error) {
	// L1: URL Parse
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidURL, err)
	}

	// L2: Scheme Check
	scheme := strings.ToLower(u.Scheme)
	allowed := false
	for _, s := range g.policy.AllowedSchemes {
		if scheme == s {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, domain.ErrSchemeNotAllowed
	}

	// Reject URLs containing userinfo (user:pass@host)
	if u.User != nil {
		return nil, fmt.Errorf("%w: URL contains userinfo", domain.ErrInvalidURL)
	}

	// L3: Hostname Blocklist
	host := strings.ToLower(u.Hostname())
	if !g.policy.AllowLocalhost {
		if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
			return nil, domain.ErrLocalhostBlocked
		}
	}
	for _, blocked := range g.policy.BlockedHosts {
		if host == blocked {
			return nil, domain.ErrSSRFBlocked
		}
	}

	// L4: DNS Resolution + Private IP Check (including IPv4-mapped IPv6)
	ips, err := net.LookupIP(host)
	if err != nil {
		// Can't resolve → allow (might be a valid hostname that resolves later)
		return u, nil
	}
	for _, ip := range ips {
		if g.isPrivateIP(ip) {
			return nil, fmt.Errorf("%w: %s resolves to %s", domain.ErrSSRFBlocked, host, ip)
		}
	}

	return u, nil
}

// isPrivateIP checks if an IP address is private/reserved.
func (g *ssrfGuard) isPrivateIP(ip net.IP) bool {
	if g.policy.AllowPrivateIPs {
		return false
	}

	// Unmap IPv4-mapped IPv6 addresses (::ffff:127.0.0.1 → 127.0.0.1)
	ip = ip.To16()
	if ip.To4() != nil {
		ip = ip.To4()
	}

	// Check private ranges
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}

	// Additional checks for IPv4
	if ip4 := ip.To4(); ip4 != nil {
		// 0.0.0.0/8 (non-routable)
		if ip4[0] == 0 {
			return true
		}
		// 169.254.0.0/16 (link-local, already covered by IsLinkLocalUnicast but explicit)
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		// 224.0.0.0/4 (multicast, covered by IsMulticast)
		// 240.0.0.0/4 (reserved)
		if ip4[0] >= 240 {
			return true
		}
	}

	return false
}
