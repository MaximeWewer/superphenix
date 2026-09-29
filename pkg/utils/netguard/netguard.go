// Package netguard checks that a user-provided URL points to a public
// destination before a platform component fetches it on the user's behalf
// (disk imports, chart repositories). It is a first line of defense: the
// fetching workload may still follow redirects or resolve the name again, so
// network isolation of that workload remains required.
package netguard

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"slices"
	"strings"
)

// Resolver resolves host names; net.DefaultResolver satisfies it.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// Options tunes a check.
type Options struct {
	// Schemes accepted, lowercase (e.g. "http", "https", "docker").
	Schemes []string
	// AllowedHosts are trusted host names or IPs that skip the destination
	// checks, for operators who import from an internal mirror on purpose.
	AllowedHosts []string
	// DeniedCIDRs are refused in addition to the non-public ranges, for
	// example public addresses of the nodes or of the management network.
	DeniedCIDRs []string
	// Resolver used for host names. Nil means net.DefaultResolver.
	Resolver Resolver
}

// Ranges refused on top of netip's private, loopback, link-local,
// multicast and unspecified checks.
var reservedPrefixes = mustPrefixes(
	"0.0.0.0/8",       // "this" network
	"100.64.0.0/10",   // carrier-grade NAT
	"192.0.0.0/24",    // IETF protocol assignments
	"192.0.2.0/24",    // documentation
	"198.18.0.0/15",   // benchmarking, used for load balancer VIPs
	"198.51.100.0/24", // documentation
	"203.0.113.0/24",  // documentation
	"240.0.0.0/4",     // reserved, includes broadcast
	"64:ff9b::/96",    // NAT64
	"64:ff9b:1::/48",  // local NAT64
	"2001:db8::/32",   // documentation
)

// Host name suffixes that designate cluster or local names.
var internalSuffixes = []string{".svc", ".cluster.local", ".local", ".localhost", ".internal", ".localdomain"}

func mustPrefixes(cidrs ...string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		prefixes = append(prefixes, netip.MustParsePrefix(c))
	}
	return prefixes
}

// CheckURL returns an error unless rawURL uses an accepted scheme and its host
// is a public destination (see package documentation).
func CheckURL(ctx context.Context, rawURL string, opts Options) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if !slices.Contains(opts.Schemes, strings.ToLower(parsed.Scheme)) {
		return fmt.Errorf("URL scheme %q is not allowed, expected one of %s", parsed.Scheme, strings.Join(opts.Schemes, ", "))
	}
	if parsed.User != nil {
		return fmt.Errorf("URL must not contain credentials")
	}

	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" {
		return fmt.Errorf("URL has no host")
	}
	if slices.Contains(opts.AllowedHosts, host) {
		return nil
	}

	denied, err := parsePrefixes(opts.DeniedCIDRs)
	if err != nil {
		return err
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		return checkAddr(addr, denied)
	}

	if host == "localhost" || !strings.Contains(host, ".") {
		return fmt.Errorf("host %q is an internal name", host)
	}
	for _, suffix := range internalSuffixes {
		if strings.HasSuffix(host, suffix) {
			return fmt.Errorf("host %q is an internal name", host)
		}
	}

	resolver := opts.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil || len(addrs) == 0 {
		return fmt.Errorf("host %q could not be resolved", host)
	}
	for _, a := range addrs {
		addr, ok := netip.AddrFromSlice(a.IP)
		if !ok {
			return fmt.Errorf("host %q resolved to an invalid address", host)
		}
		if err := checkAddr(addr, denied); err != nil {
			return fmt.Errorf("host %q: %w", host, err)
		}
	}
	return nil
}

func checkAddr(addr netip.Addr, denied []netip.Prefix) error {
	addr = addr.Unmap()
	if !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() || addr.IsMulticast() || addr.IsUnspecified() {
		return fmt.Errorf("address %s is not a public address", addr)
	}
	for _, p := range reservedPrefixes {
		if p.Contains(addr) {
			return fmt.Errorf("address %s is in reserved range %s", addr, p)
		}
	}
	for _, p := range denied {
		if p.Contains(addr) {
			return fmt.Errorf("address %s is in denied range %s", addr, p)
		}
	}
	return nil
}

func parsePrefixes(cidrs []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(strings.TrimSpace(c))
		if err != nil {
			return nil, fmt.Errorf("invalid denied CIDR %q: %w", c, err)
		}
		prefixes = append(prefixes, p.Masked())
	}
	return prefixes, nil
}
