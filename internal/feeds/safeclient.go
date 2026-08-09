package feeds

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// errBlockedAddress is returned when a feed fetch (or any hop it redirects
// through) tries to connect to a non-public IP.
var errBlockedAddress = errors.New("connection to a private or non-public address is not allowed")

// NewSafeHTTPClient builds the HTTP client the feed worker and image probes use.
// Feed endpoints are admin-supplied URLs fetched server-side, so an unrestricted
// client is an SSRF primitive: it would reach cloud metadata (169.254.169.254),
// localhost, and RFC1918 hosts. This client refuses to *connect* to any non-public
// IP, enforced in the dialer's Control hook so it applies to the resolved address
// of every connection — the initial request and every redirect hop alike — closing
// the DNS-rebinding gap a URL-string check alone would leave open. It also caps
// redirects.
func NewSafeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || !isPublicIP(ip) {
				return fmt.Errorf("%w: %s", errBlockedAddress, host)
			}
			return nil
		},
	}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("stopped after 5 redirects")
			}
			return nil
		},
	}
}

// isPublicIP reports whether ip is a globally routable unicast address, rejecting
// loopback, private (RFC1918 / ULA), link-local (incl. the 169.254.169.254 cloud
// metadata endpoint), and the unspecified address.
func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	return true
}
