// Network contains a default implementation of the network.
//
// Please see [mtglib.Network] interface to get some basic idea behind this
// abstraction.
//
// Some notable feature of this implementation:
//
//  1. It detaches dialer from a network. Dialer is something which implements a
//     real dialer and network completes it with more higher level details.
//  2. It uses only TCP connections. Even for DNS it uses DNS-Over-HTTPS
//  3. It has some simple implementation of DNS cache which is good enough for
//     our purpose.
//  4. It sets uses SO_REUSEPORT port if applicable.
package network

import (
	"context"
	"errors"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
)

const (
	// DefaultTimeout is a default timeout for establishing TCP connection.
	DefaultTimeout = 10 * time.Second

	// DefaultHTTPTimeout defines a default timeout for making HTTP request.
	DefaultHTTPTimeout = 10 * time.Second

	// Deprecated:
	//
	// DefaultBufferSize defines a TCP buffer size. Both read and write, so for
	// real size, please multiply this number by 2.
	DefaultBufferSize = 16 * 1024 // 16 kib

	// DefaultTCPKeepAlivePeriod defines a time period between 2 consequitive
	// probes.
	DefaultTCPKeepAlivePeriod = 10 * time.Second

	// ProxyDialerOpenThreshold is used for load balancing SOCKS5 dialer only.
	//
	// This dialer uses circuit breaker with of 3 stages: OPEN, HALF_OPEN and
	// CLOSED. If state is CLOSED, all requests go in a normal mode. If you get
	// more that ProxyDialerOpenThreshold errors, circuit breaker goes into OPEN
	// mode.
	//
	// When circuit breaker is in OPEN mode, it forbids all request to a given
	// proxy. But after ProxyDialerHalfOpenTimeout it gives a second chance and
	// opens an access for a SINGLE request. If this request success, then circuit
	// breaker closes, otherwise opens again.
	//
	// When circuit breaker is closed, it clears an error states each
	// ProxyDialerResetFailuresTimeout.
	ProxyDialerOpenThreshold = 5

	// ProxyDialerHalfOpenTimeout defines a halfopen timeout for circuit breaker.
	ProxyDialerHalfOpenTimeout = time.Minute

	// ProxyDialerResetFailuresTimeout defines a timeout for resetting a failure.
	ProxyDialerResetFailuresTimeout = 10 * time.Second

	// DefaultDOHHostname defines a default IP address for DOH host. Since mtg is
	// simple, please pass IP address here. We do not have bootstrap servers here
	// embedded.
	DefaultDOHHostname = "9.9.9.9"

	// DNSTimeout defines a timeout for DNS queries.
	DNSTimeout = 5 * time.Second

	// tcpLingerTimeout defines a number of seconds to wait for sending
	// unacknowledged data.
	tcpLingerTimeout = 1
)

var (
	// ErrCircuitBreakerOpened is returned when proxy is being accessed but
	// circuit breaker is opened.
	ErrCircuitBreakerOpened = errors.New("circuit breaker is opened")

	// ErrCannotDialWithAllProxies is returned when load balancing client is
	// trying to access proxies but all of them are failed.
	ErrCannotDialWithAllProxies = errors.New("cannot dial with all proxies")
)

// Dialer defines an interface which is required to bootstrap a network
// instance from.
type Dialer interface {
	Dial(network, address string) (essentials.Conn, error)
	DialContext(ctx context.Context, network, address string) (essentials.Conn, error)
}
