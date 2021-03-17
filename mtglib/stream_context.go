package mtglib

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net"
	"time"
)

type streamContext struct {
	ctx        context.Context
	ctxCancel  context.CancelFunc
	clientConn net.Conn
	connID     string
	logger     Logger
}

func (s *streamContext) Deadline() (time.Time, bool) {
	return s.ctx.Deadline()
}

func (s *streamContext) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *streamContext) Err() error {
	return s.ctx.Err()
}

func (s *streamContext) Value(key interface{}) interface{} {
	return s.ctx.Value(key)
}

func (s *streamContext) Close() {
	s.ctxCancel()
	s.clientConn.Close()
}

func (s *streamContext) ClientIP() net.IP {
	return s.clientConn.RemoteAddr().(*net.TCPAddr).IP
}

func newStreamContext(ctx context.Context, logger Logger, clientConn net.Conn) *streamContext {
	connIDBytes := make([]byte, 16)

	if _, err := rand.Read(connIDBytes); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(ctx)
	streamCtx := &streamContext{
		ctx:        ctx,
		ctxCancel:  cancel,
		clientConn: clientConn,
		connID:     base64.RawURLEncoding.EncodeToString(connIDBytes),
	}
	streamCtx.logger = logger.
		BindStr("stream-id", streamCtx.connID).
		BindStr("client-ip", streamCtx.ClientIP().String())

	return streamCtx
}
