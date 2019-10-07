package proxy

import (
	"context"
	"io"
	"net"
	"sync"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/protocol"
	"github.com/9seconds/mtg/stats"
	"github.com/9seconds/mtg/utils"
	"github.com/9seconds/mtg/wrappers"
)

const directPipeBufferSize = 1024 * 1024

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
			stats.S.Crash()
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

	clientConn := wrappers.NewClientConn(conn, connID)
	clientConn = wrappers.NewCtx(ctx, cancel, clientConn)
	clientConn = wrappers.NewTimeout(clientConn)
	clientConn = wrappers.NewTraffic(clientConn)
	defer clientConn.Close()

	clientProtocol := p.ClientProtocolMaker()
	clientConn, err := clientProtocol.Handshake(clientConn)
	if err != nil {
		logger.Warnw("Cannot perform client handshake", "error", err)
		return
	}

	stats.S.ClientConnected(clientProtocol.ConnectionType(), clientConn.RemoteAddr())
	defer stats.S.ClientDisconnected(clientProtocol.ConnectionType(), clientConn.RemoteAddr())
	logger.Infow("Client connected", "addr", conn.RemoteAddr())

	req := &protocol.TelegramRequest{
		Logger:         logger,
		ClientConn:     clientConn,
		ConnID:         connID,
		Ctx:            ctx,
		Cancel:         cancel,
		ClientProtocol: clientProtocol,
	}

	if len(config.C.AdTag) > 0 {
		err = p.acceptMiddleProxyConnection(req)
	} else {
		err = p.acceptDirectConnection(req)
	}

	logger.Infow("Client disconnected", "error", err, "addr", conn.RemoteAddr())
}

func (p *Proxy) acceptDirectConnection(request *protocol.TelegramRequest) error {
	telegramConnRaw, err := obfuscated2.TelegramProtocol(request)
	if err != nil {
		return err
	}
	telegramConn := telegramConnRaw.(conntypes.StreamReadWriteCloser)
	defer telegramConn.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go p.directPipe(telegramConn, request.ClientConn, wg, request.Logger)
	go p.directPipe(request.ClientConn, telegramConn, wg, request.Logger)

	<-request.Ctx.Done()
	wg.Wait()

	return request.Ctx.Err()
}

func (p *Proxy) directPipe(dst io.Writer,
	src io.Reader,
	wg *sync.WaitGroup,
	logger *zap.SugaredLogger) {
	defer wg.Done()

	buf := make([]byte, directPipeBufferSize)
	if _, err := io.CopyBuffer(dst, src, buf); err != nil {
		logger.Debugw("Cannot pump sockets", "error", err)
	}
}

func (p *Proxy) acceptMiddleProxyConnection(request *protocol.TelegramRequest) error {
	return nil
}
