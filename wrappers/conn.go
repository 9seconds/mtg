package wrappers

import (
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/conntypes"
)

type connPurpose uint8

const (
	connPurposeClient connPurpose = 1 << iota
	connPurposeTelegram
)

type wrapperConn struct {
	parent     net.Conn
	connID     conntypes.ConnID
	logger     *zap.SugaredLogger
	localAddr  *net.TCPAddr
	remoteAddr *net.TCPAddr
}

func (w *wrapperConn) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	if err := w.parent.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		w.Close()
		return 0, fmt.Errorf("cannot set write deadline to the socket: %w", err)
	}

	return w.Write(p)
}

func (w *wrapperConn) Write(p []byte) (int, error) {
	n, err := w.parent.Write(p)
	w.logger.Debugw("write to stream", "bytes", n, "error", err)
	if err != nil {
		w.Close() // nolint: gosec
	}

	return n, err
}

func (w *wrapperConn) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	if err := w.parent.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		w.Close()
		return 0, fmt.Errorf("cannot set read deadline to the socket: %w", err)
	}

	return w.Read(p)
}

func (w *wrapperConn) Read(p []byte) (int, error) {
	n, err := w.parent.Read(p)
	w.logger.Debugw("Read from stream", "bytes", n, "error", err)
	if err != nil {
		w.Close()
	}

	return n, err
}

func (w *wrapperConn) Close() error {
	w.logger.Debugw("Close connection")
	return w.parent.Close()
}

func (w *wrapperConn) Conn() net.Conn {
	return w.parent
}

func (w *wrapperConn) Logger() *zap.SugaredLogger {
	return w.logger
}

func (w *wrapperConn) LocalAddr() *net.TCPAddr {
	return w.localAddr
}

func (w *wrapperConn) RemoteAddr() *net.TCPAddr {
	return w.remoteAddr
}

func newConn(parent net.Conn,
	connID conntypes.ConnID,
	purpose connPurpose) conntypes.StreamReadWriteCloser {
	localAddr := *parent.LocalAddr().(*net.TCPAddr)

	if parent.RemoteAddr().(*net.TCPAddr).IP.To4() != nil {
		if config.C.PublicIPv4Addr.IP != nil {
			localAddr.IP = config.C.PublicIPv4Addr.IP
		}
	} else if config.C.PublicIPv6Addr.IP != nil {
		localAddr.IP = config.C.PublicIPv6Addr.IP
	}

	logger := zap.S().With(
		"local_address", localAddr,
		"remote_address", parent.RemoteAddr(),
	).Named("conn")

	if purpose == connPurposeClient {
		logger = logger.With("connection_id", connID.String())
	}

	return &wrapperConn{
		parent:     parent,
		connID:     connID,
		logger:     logger,
		remoteAddr: parent.RemoteAddr().(*net.TCPAddr),
		localAddr:  &localAddr,
	}
}

func NewClientConn(parent net.Conn,
	connID conntypes.ConnID) conntypes.StreamReadWriteCloser {
	return newConn(parent, connID, connPurposeClient)
}

func NewTelegramConn(parent net.Conn) conntypes.StreamReadWriteCloser {
	return newConn(parent, conntypes.ConnID{}, connPurposeTelegram)
}
