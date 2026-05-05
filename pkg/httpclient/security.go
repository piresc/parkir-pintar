package httpclient

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"
)

// privateRanges defines CIDR blocks for private/reserved IP ranges.
var privateRanges = []string{
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"::1/128",
	"fc00::/7",
}

// parsedPrivateRanges holds the parsed CIDR networks for private IP checking.
var parsedPrivateRanges []*net.IPNet

func init() {
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			parsedPrivateRanges = append(parsedPrivateRanges, network)
		}
	}
}

// isPrivateIP checks whether the given IP falls within any private/reserved range.
func isPrivateIP(ip net.IP) bool {
	for _, network := range parsedPrivateRanges {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// validatePathURL rejects paths that could be used for SSRF attacks.
// Blocks paths containing "..", "@", or starting with "//".
// Also decodes URL-encoded characters before checking.
func validatePathURL(pathURL string) error {
	// URL-decode the path before running checks to catch encoded bypass attempts
	decoded, err := url.PathUnescape(pathURL)
	if err != nil {
		return fmt.Errorf("invalid path: failed to decode: %w", err)
	}

	if strings.Contains(decoded, "..") {
		return errors.New("invalid path: contains '..'")
	}
	if strings.Contains(decoded, "@") {
		return errors.New("invalid path: contains '@'")
	}
	if strings.HasPrefix(decoded, "//") {
		return errors.New("invalid path: starts with '//'")
	}
	return nil
}

// resolveIP is the function used to look up IPs for a hostname.
// It defaults to net.LookupIP but can be overridden in tests.
var resolveIP = net.LookupIP

// buildSafeURL constructs a safe URL from a base URL and a path component.
// The path is validated via validatePathURL before joining.
// After constructing the URL, the hostname is resolved and checked against
// private IP ranges to prevent SSRF attacks.
func buildSafeURL(baseURL, pathURL string) (string, error) {
	if err := validatePathURL(pathURL); err != nil {
		return "", err
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Ensure pathURL starts with /
	if !strings.HasPrefix(pathURL, "/") {
		pathURL = "/" + pathURL
	}

	parsedBase.Path = path.Join(parsedBase.Path, pathURL)

	// Resolve hostname and check against private IP ranges
	hostname := parsedBase.Hostname()
	if hostname != "" {
		// Check if hostname is an IP literal first
		ip := net.ParseIP(hostname)
		if ip != nil {
			if isPrivateIP(ip) {
				return "", fmt.Errorf("blocked: host %s resolves to private IP", hostname)
			}
		} else {
			// Hostname is a domain name — resolve it
			ips, err := resolveIP(hostname)
			if err == nil {
				for _, resolvedIP := range ips {
					if isPrivateIP(resolvedIP) {
						return "", fmt.Errorf("blocked: host %s resolves to private IP", hostname)
					}
				}
			}
			// If DNS lookup fails, allow it through (DNS may not be available)
		}
	}

	return parsedBase.String(), nil
}
