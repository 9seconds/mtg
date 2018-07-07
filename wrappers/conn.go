package wrappers

import (
	"net"
	"time"

	"go.uber.org/zap"
)

type ConnPurpose uint8

func (c ConnPurpose) String() string {
	switch c {
	case ConnPurposeClient:
		return "client"
	case ConnPurposeTelegram:
		return "telegram"
	}

	return ""
}

const (
	ConnPurposeClient = iota
	ConnPurposeTelegram
)

const (
	connTimeoutRead  = 5 * time.Minute
	connTimeoutWrite = 5 * time.Minute
)

type WrapConn struct {
	purpose    ConnPurpose
	connID     string
	conn       net.Conn
	logger     *zap.SugaredLogger
	publicIPv4 net.IP
	publicIPv6 net.IP
}

func (w *WrapConn) Write(p []byte) (int, error) {
	w.conn.SetWriteDeadline(time.Now().Add(connTimeoutWrite))
	n, err := w.conn.Write(p)

	w.logger.Debugw("Write to stream", "bytes", n, "error", err)

	return n, err
}

func (w *WrapConn) Read(p []byte) (int, error) {
	w.conn.SetReadDeadline(time.Now().Add(connTimeoutRead))
	n, err := w.conn.Read(p)

	w.logger.Debugw("Read from stream", "bytes", n, "error", err)

	return n, err
}

func (w *WrapConn) Close() error {
	defer w.LogDebug("Closed connection")
	return w.conn.Close()
}

func (w *WrapConn) LocalAddr() *net.TCPAddr {
	addr := w.conn.LocalAddr().(*net.TCPAddr)
	newAddr := *addr

	if w.RemoteAddr().IP.To4() != nil {
		if w.publicIPv4 != nil {
			newAddr.IP = w.publicIPv4
		}
	} else if w.publicIPv6 != nil {
		newAddr.IP = w.publicIPv6
	}

	return &newAddr
}

func (w *WrapConn) RemoteAddr() *net.TCPAddr {
	return w.conn.RemoteAddr().(*net.TCPAddr)
}

func (w *WrapConn) LogDebug(msg string, data ...interface{}) {
	w.logger.Debugw(msg, data...)
}

func (w *WrapConn) LogInfo(msg string, data ...interface{}) {
	w.logger.Infow(msg, data...)
}

func (w *WrapConn) LogWarn(msg string, data ...interface{}) {
	w.logger.Warnw(msg, data...)
}

func (w *WrapConn) LogError(msg string, data ...interface{}) {
	w.logger.Errorw(msg, data...)
}

func NewConn(connID string, purpose ConnPurpose, conn net.Conn, publicIPv4, publicIPv6 net.IP) WrapStreamReadWriteCloser {
	logger := zap.S().With(
		"connection_id", connID,
		"local_address", conn.LocalAddr(),
		"remote_address", conn.RemoteAddr(),
	)

	return &WrapConn{
		logger:     logger,
		purpose:    purpose,
		connID:     connID,
		conn:       conn,
		publicIPv4: publicIPv4,
		publicIPv6: publicIPv6,
	}
}
