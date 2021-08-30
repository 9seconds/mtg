package network

import (
	"context"
	"fmt"
	"net"
	"time"
)

type defaultDialer struct {
	net.Dialer

	bufferSize int
}

func (d *defaultDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *defaultDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	switch network {
	case "tcp", "tcp4", "tcp6": // nolint: goconst
	default:
		return nil, fmt.Errorf("unsupported network %s", network)
	}

	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("cannot dial to %s: %w", address, err)
	}

	// we do not need to call to end user. End users call us.
	if err := SetServerSocketOptions(conn, d.bufferSize); err != nil {
		conn.Close()

		return nil, fmt.Errorf("cannot set socket options: %w", err)
	}

	return conn, nil
}

// NewDefaultDialer build a new dialer which dials bypassing proxies
// etc.
//
// The most default one you can imagine. But it has tunes TCP
// connections and setups SO_REUSEPORT.
func NewDefaultDialer(timeout time.Duration, bufferSize int) (Dialer, error) {
	switch {
	case timeout < 0:
		return nil, fmt.Errorf("timeout %v should be positive number", timeout)
	case bufferSize < 0:
		return nil, fmt.Errorf("buffer size %d should be positive number", bufferSize)
	}

	if timeout == 0 {
		timeout = DefaultTimeout
	}

	if bufferSize == 0 {
		bufferSize = DefaultBufferSize
	}

	return &defaultDialer{
		Dialer: net.Dialer{
			Timeout: timeout,
		},
		bufferSize: bufferSize,
	}, nil
}
