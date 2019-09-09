package wrappers

import (
	"context"
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

const (
	connTimeoutRead  = 2 * time.Minute
	connTimeoutWrite = 2 * time.Minute
)

type wrapperConn struct {
	parent     net.Conn
	ctx        context.Context
	cancel     context.CancelFunc
	connID     conntypes.ConnID
	logger     *zap.SugaredLogger
	localAddr  *net.TCPAddr
	remoteAddr *net.TCPAddr
}

func (w *wrapperConn) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	select {
	case <-w.ctx.Done():
		w.Close()
		return 0, fmt.Errorf("cannot write because context was closed: %w", w.ctx.Err())

	default:
		if err := w.parent.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			w.Close() // nolint: gosec
			return 0, fmt.Errorf("cannot set write deadline to the socket: %w", err)
		}

		n, err := w.parent.Write(p)
		w.logger.Debugw("Write to stream", "bytes", n, "error", err)
		if err != nil {
			w.Close() // nolint: gosec
		}

		return n, err
	}
}

func (w *wrapperConn) Write(p []byte) (int, error) {
	return w.WriteTimeout(p, connTimeoutWrite)
}

func (w *wrapperConn) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	select {
	case <-w.ctx.Done():
		w.Close()
		return 0, fmt.Errorf("cannot read because context was closed: %w", w.ctx.Err())

	default:
		if err := w.parent.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			w.Close()
			return 0, fmt.Errorf("cannot set read deadline to the socket: %w", err)
		}

		n, err := w.parent.Read(p)
		w.logger.Debugw("Read from stream", "bytes", n, "error", err)
		if err != nil {
			w.Close()
		}

		return n, err
	}
}

func (w *wrapperConn) Read(p []byte) (int, error) {
	return w.ReadTimeout(p, connTimeoutRead)
}

func (w *wrapperConn) Close() error {
	w.logger.Debugw("Close connection")
	w.cancel()

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

func newConn(ctx context.Context,
	cancel context.CancelFunc,
	parent net.Conn,
	connID conntypes.ConnID,
	purpose connPurpose) StreamReadWriteCloser {
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
		ctx:        ctx,
		cancel:     cancel,
		connID:     connID,
		logger:     logger,
		remoteAddr: parent.RemoteAddr().(*net.TCPAddr),
		localAddr:  &localAddr,
	}
}

func NewClientConn(ctx context.Context,
	cancel context.CancelFunc,
	parent net.Conn,
	connID conntypes.ConnID) StreamReadWriteCloser {
	return newConn(ctx, cancel, parent, connID, connPurposeClient)
}

func NewTelegramConn(ctx context.Context,
	cancel context.CancelFunc,
	parent net.Conn) StreamReadWriteCloser {
	return newConn(ctx, cancel, parent, conntypes.ConnID{}, connPurposeTelegram)
}
