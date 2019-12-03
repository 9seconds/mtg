package proxy

import (
	"context"
	"net"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/protocol"
	"github.com/9seconds/mtg/stats"
	"github.com/9seconds/mtg/utils"
	"github.com/9seconds/mtg/wrappers/stream"
)

type Proxy struct {
	Logger              *zap.SugaredLogger
	Context             context.Context
	ClientProtocolMaker protocol.ClientProtocolMaker
}

func (p *Proxy) Serve(listener net.Listener) {
	doneChan := p.Context.Done()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-doneChan:
				return
			default:
				p.Logger.Errorw("Cannot allocate incoming connection", "error", err)
				continue
			}
		}

		go p.accept(conn)
	}
}

func (p *Proxy) accept(conn net.Conn) {
	defer func() {
		conn.Close()
		if err := recover(); err != nil {
			stats.Stats.Crash()
			p.Logger.Errorw("Crash of accept handler", "error", err)
		}
	}()

	connID := conntypes.NewConnID()
	logger := p.Logger.With("connection_id", connID)

	if err := utils.InitTCP(conn); err != nil {
		logger.Errorw("Cannot initialize client TCP connection", "error", err)
		return
	}

	ctx, cancel := context.WithCancel(p.Context)
	defer cancel()

	clientConn := stream.NewClientConn(conn, connID)
	clientConn = stream.NewCtx(ctx, cancel, clientConn)
	clientConn = stream.NewTimeout(clientConn)

	defer clientConn.Close()

	clientProtocol := p.ClientProtocolMaker()
	clientConn, err := clientProtocol.Handshake(clientConn)

	if err != nil {
		stats.Stats.AuthenticationFailed()
		logger.Warnw("Cannot perform client handshake", "error", err)

		return
	}

	stats.Stats.ClientConnected(clientProtocol.ConnectionType(), clientConn.RemoteAddr())
	defer stats.Stats.ClientDisconnected(clientProtocol.ConnectionType(), clientConn.RemoteAddr())
	logger.Infow("Client connected", "addr", conn.RemoteAddr())

	req := &protocol.TelegramRequest{
		Logger:         logger,
		ClientConn:     clientConn,
		ConnID:         connID,
		Ctx:            ctx,
		Cancel:         cancel,
		ClientProtocol: clientProtocol,
	}

	err = nil

	if len(config.C.AdTag) > 0 {
		middleConnection(req)
	} else {
		err = directConnection(req)
	}

	logger.Infow("Client disconnected", "error", err, "addr", conn.RemoteAddr())
}
