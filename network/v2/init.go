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
	//
	// Deprecated: use DefaultKeepAliveConfig
	DefaultTCPKeepAlivePeriod = 10 * time.Second

	// DefaultKeepAliveIdle is the time a connection must be idle before
	// the first keepalive probe is sent.
	//
	// Deprecated: use DefaultKeepAliveConfig
	DefaultKeepAliveIdle = 30 * time.Second

	// DefaultKeepAliveInterval is the time between consecutive keepalive
	// probes.
	//
	// Deprecated: use DefaultKeepAliveConfig
	DefaultKeepAliveInterval = 10 * time.Second

	// DefaultKeepAliveCount is the number of unacknowledged probes before
	// the connection is considered dead.
	//
	// Deprecated: use DefaultKeepAliveConfig
	DefaultKeepAliveCount = 3

	// User Agent to use in HTTP client.
	UserAgent = "curl/8.5.0"

	// tcpLingerTimeout defines a number of seconds to wait for sending
	// unacknowledged data.
	tcpLingerTimeout = 1

	// tcpNotSentLowat limits the amount of unsent data queued in the
	// kernel write buffer per socket. When the unsent data drops below
	// this threshold, the socket becomes writable again. This reduces
	// per-connection memory usage and bufferbloat by applying
	// back-pressure to the relay loop instead of piling up data in
	// kernel buffers.
	tcpNotSentLowat = 128 * 1024
)

var (
	ErrCannotDial = errors.New("cannot dial to any address")

	// DefaultKeepAliveConfig defines a default configuration for
	// keep alive settings. As per official documentation, if keep alive
	// is enabled, then:
	//
	//  Idle = 15 * time.Second
	//  Interval = 15 * time.Second
	//  Count = 9
	DefaultKeepAliveConfig = net.KeepAliveConfig{
		Enable: true,
	}
)
