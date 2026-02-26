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
	"net"
	"time"

	"github.com/9seconds/mtg/v2/mtglib"
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

	// tcpLingerTimeout defines a number of seconds to wait for sending
	// unacknowledged data.
	tcpLingerTimeout = 1
)

var ErrCannotDial = errors.New("cannot dial to any address")

type Network interface {
	mtglib.Network

	NativeDialer() *net.Dialer
}
