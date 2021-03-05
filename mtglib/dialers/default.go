package dialers

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-reuseport"
)

type defaultBaseDialer struct {
	net.Dialer

	bufferSize int
}

func (d *defaultBaseDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *defaultBaseDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, fmt.Errorf("unsupported network %s", network)
	}

	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("cannot dial to %s: %w", address, err)
	}

	tcpConn := conn.(*net.TCPConn)

	if err := tcpConn.SetNoDelay(true); err != nil {
		conn.Close()
		return nil, fmt.Errorf("cannot set TCP_NO_DELAY: %w", err)
	}

	if err := tcpConn.SetReadBuffer(d.bufferSize); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("cannot set read buffer size: %w", err)
	}

	if err := tcpConn.SetWriteBuffer(d.bufferSize); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("cannot set write buffer size: %w", err)
	}

	if err := tcpConn.SetKeepAlive(true); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("cannot enable keep-alive: %w", err)
	}

	return tcpConn, nil
}

func NewDefaultBaseDialer(timeout time.Duration, bufferSize int) (BaseDialer, error) {
	switch {
	case timeout < 0:
		return nil, fmt.Errorf("timeout %v should be positive number", timeout)
	case bufferSize < 0:
		return nil, fmt.Errorf("buffer size %s should be positive number", bufferSize)
	}

	if timeout == 0 {
		timeout = DefaultTimeout
	}

	if bufferSize == 0 {
		bufferSize = DefaultBufferSize
	}

	return &defaultBaseDialer{
		Dialer: net.Dialer{
			Timeout: timeout,
			Control: reuseport.Control,
		},
		bufferSize: bufferSize,
	}, nil
}
