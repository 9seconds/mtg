package mtglib

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
)

type streamContext struct {
	ctx          context.Context
	ctxCancel    context.CancelFunc
	clientConn   essentials.Conn
	telegramConn essentials.Conn
	streamID     string
	dc           int
	logger       Logger
}

func (s *streamContext) Deadline() (time.Time, bool) {
	return s.ctx.Deadline()
}

func (s *streamContext) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *streamContext) Err() error {
	return s.ctx.Err() //nolint: wrapcheck
}

func (s *streamContext) Value(key interface{}) interface{} {
	return s.ctx.Value(key)
}

func (s *streamContext) Close() {
	s.ctxCancel()

	if s.clientConn != nil {
		s.clientConn.Close()
	}

	if s.telegramConn != nil {
		s.telegramConn.Close()
	}
}

func (s *streamContext) ClientIP() net.IP {
	return s.clientConn.RemoteAddr().(*net.TCPAddr).IP //nolint: forcetypeassert
}

func newStreamContext(ctx context.Context, logger Logger, clientConn essentials.Conn) *streamContext {
	connIDBytes := make([]byte, ConnectionIDBytesLength)

	if _, err := rand.Read(connIDBytes); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(ctx)
	streamCtx := &streamContext{
		ctx:        ctx,
		ctxCancel:  cancel,
		clientConn: clientConn,
		streamID:   base64.RawURLEncoding.EncodeToString(connIDBytes),
	}
	streamCtx.logger = logger.
		BindStr("stream-id", streamCtx.streamID).
		BindStr("client-ip", streamCtx.ClientIP().String())

	return streamCtx
}
