package utils

import (
	"errors"
	"fmt"
	"net"

	"github.com/9seconds/mtg/v2/network"
)

var errListenerNotTCP = errors.New("listener is not TCP")

// SetListenerMSS sets TCP_MAXSEG on the listening socket of a listener made by
// NewListener (Linux only, a no-op elsewhere). Connections accepted after that
// inherit it: the SYN-ACK advertises this MSS to the client, and the kernel
// never sends the client segments larger than that for the whole connection.
// mss <= 0 leaves the socket alone.
//
// It is applied to an already listening socket on purpose, so that
// NewListener keeps its signature. Call it right after NewListener, before the
// listener is wrapped (PROXY protocol) and before Serve: a client whose SYN
// arrives in between gets the default MSS.
func SetListenerMSS(listener net.Listener, mss int) error {
	if mss <= 0 {
		return nil
	}

	tcp, ok := tcpListenerOf(listener)
	if !ok {
		return fmt.Errorf("cannot set listener MSS: %w", errListenerNotTCP)
	}

	rawConn, err := tcp.SyscallConn()
	if err != nil {
		return fmt.Errorf("cannot access listener socket: %w", err)
	}

	addr := tcp.Addr()

	return network.ListenControlMSS(mss)(addr.Network(), addr.String(), rawConn) //nolint: wrapcheck
}

// tcpListenerOf returns the TCP listener under a listener of this package or
// a plain *net.TCPListener.
func tcpListenerOf(listener net.Listener) (*net.TCPListener, bool) {
	switch l := listener.(type) {
	case *net.TCPListener:
		return l, true
	case interface{ base() net.Listener }:
		return tcpListenerOf(l.base())
	}

	return nil, false
}

// base returns the wrapped listener. The value receiver makes it available on
// both Listener and *Listener.
func (l Listener) base() net.Listener {
	return l.Listener
}
