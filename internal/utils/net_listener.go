package utils

import (
	"fmt"
	"net"
	"time"

	"github.com/9seconds/mtg/v2/internal/logthrottle"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/network"
)

// socketOptionsLogInterval limits the warning about connections dropped
// because their socket options cannot be set: one line per interval, with the
// number of suppressed ones.
const socketOptionsLogInterval = 10 * time.Second

// Listener applies client socket options to accepted connections.
type Listener struct {
	net.Listener

	logger   mtglib.Logger
	throttle *logthrottle.Throttle
}

// WrapListener returns a Listener that accepts connections from base.
func WrapListener(base net.Listener, logger mtglib.Logger) *Listener {
	return &Listener{
		Listener: base,
		logger:   logger,
		throttle: logthrottle.New(socketOptionsLogInterval),
	}
}

// Accept returns the next connection with client socket options applied. A
// connection whose options cannot be set (typically it was reset by the client
// right after the handshake) is closed and skipped: returning an error here
// would stop the whole accept loop because of one bad client.
//
// The skip is logged as a warning, rate limited: if setting the options starts
// failing for every connection (a bad option, an unsupported platform), the
// proxy must not drop everything silently, nor flood the log under load.
func (l *Listener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err //nolint: wrapcheck
		}

		if err := network.SetClientSocketOptions(conn, 0); err != nil {
			l.logSkipped(conn, err)
			conn.Close() //nolint: errcheck

			continue
		}

		return conn, nil
	}
}

func (l *Listener) logSkipped(conn net.Conn, err error) {
	ok, suppressed := l.throttle.Allow()
	if !ok {
		return
	}

	logger := l.logger.BindInt("suppressed", suppressed)

	if addr := conn.RemoteAddr(); addr != nil {
		logger = logger.BindStr("ip", addr.String())
	}

	logger.WarningError("cannot set client socket options, the connection is dropped", err)
}

func NewListener(bindTo string, bufferSize int, logger mtglib.Logger) (net.Listener, error) {
	base, err := net.Listen("tcp", bindTo)
	if err != nil {
		return nil, fmt.Errorf("cannot build a base listener: %w", err)
	}

	return WrapListener(base, logger), nil
}
