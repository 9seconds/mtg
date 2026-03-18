// Network contains a default implementation of the network.
//
// Please see [mtglib.Network] interface to get some basic idea behind this
// abstraction.
//
// This implementation is more simple that v1 because life shows that all
// this complexity, especially around circuit breakers and DoH is not really
// required. There is no chance that if DNS address is spoofed, that real
// IP would work as expected.
package network

import (
	"errors"
	"time"
)

const (
	// DefaultTimeout is a default timeout for establishing TCP connection.
	DefaultTimeout = 10 * time.Second

	// DefaultHTTPTimeout defines a default timeout for making HTTP request.
	DefaultHTTPTimeout = 10 * time.Second

	// DefaultIdleTimeout defines a timeout for idle HTTP connections
	DefaultIdleTimeout = time.Minute

	// DefaultTCPKeepAlivePeriod defines a time period between 2 consecuitive
	// probes.
	DefaultTCPKeepAlivePeriod = 10 * time.Second

	// User Agent to use in HTTP client.
	UserAgent = "curl/8.5.0"

	// tcpLingerTimeout defines a number of seconds to wait for sending
	// unacknowledged data.
	tcpLingerTimeout = 1
)

type TLSProfile string

const (
	TLSProfileChrome  TLSProfile = "chrome"
	TLSProfileFirefox TLSProfile = "firefox"
	TLSProfileSafari  TLSProfile = "safari"
	TLSProfileEdge    TLSProfile = "edge"
)

const DefaultTLSProfile = TLSProfileChrome

// Last updated: 2026-03.
var tlsProfileUserAgents = map[TLSProfile]string{
	TLSProfileChrome:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	TLSProfileFirefox: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
	TLSProfileSafari:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7_2) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Safari/605.1.15",
	TLSProfileEdge:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
}

func GetTLSProfileUserAgent(profile TLSProfile, fallback string) string {
	if ua, ok := tlsProfileUserAgents[profile]; ok {
		return ua
	}

	return fallback
}

func ValidTLSProfile(profile TLSProfile) bool {
	switch profile {
	case TLSProfileChrome, TLSProfileFirefox, TLSProfileSafari, TLSProfileEdge:
		return true
	default:
		return false
	}
}

var ErrCannotDial = errors.New("cannot dial to any address")
