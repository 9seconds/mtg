package proxy

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/juju/errors"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"

	"github.com/9seconds/mtg/client"
	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/telegram"
	"github.com/9seconds/mtg/wrappers"
)

type Proxy struct {
	clientInit client.Init
	tg         telegram.Telegram
	conf       *config.Config
}

func (p *Proxy) Serve() error {
	lsock, err := net.Listen("tcp", p.conf.BindAddr())
	if err != nil {
		return errors.Annotate(err, "Cannot create listen socket")
	}

	for {
		if conn, err := lsock.Accept(); err != nil {
			zap.S().Errorw("Cannot allocate incoming connection", "error", err)
		} else {
			go p.accept(conn)
		}
	}
}

func (p *Proxy) accept(conn net.Conn) {
	connID := uuid.NewV4().String()
	log := zap.S().With("connection_id", connID)

	defer func() {
		conn.Close()

		if err := recover(); err != nil {
			log.Errorw("Crash of accept handler", "error", err)
		}
	}()

	log.Infow("Client connected", "addr", conn.RemoteAddr())

	ctx, cancel := context.WithCancel(context.Background())
	client, opts, err := p.clientInit(ctx, cancel, conn, connID, p.conf)
	if err != nil {
		log.Errorw("Cannot initialize client connection", "error", err)
		return
	}
	defer client.(wrappers.WrapCloser).Close()

	server, err := p.getTelegramConn(ctx, cancel, opts, connID)
	if err != nil {
		log.Errorw("Cannot initialize server connection", "error", err)
		return
	}
	defer server.(wrappers.WrapCloser).Close()

	wait := &sync.WaitGroup{}
	wait.Add(2)

	if p.conf.UseMiddleProxy() {
		clientPacket := client.(wrappers.WrapPacketReadWriteCloser)
		serverPacket := server.(wrappers.WrapPacketReadWriteCloser)
		go p.middlePipe(clientPacket, serverPacket, wait, &opts.ReadHacks)
		go p.middlePipe(serverPacket, clientPacket, wait, &opts.WriteHacks)
	} else {
		clientStream := client.(wrappers.WrapStreamReadWriteCloser)
		serverStream := server.(wrappers.WrapStreamReadWriteCloser)
		go p.directPipe(clientStream, serverStream, wait)
		go p.directPipe(serverStream, clientStream, wait)
	}

	<-ctx.Done()
	wait.Wait()

	log.Infow("Client disconnected", "addr", conn.RemoteAddr())
}

func (p *Proxy) getTelegramConn(ctx context.Context, cancel context.CancelFunc, opts *mtproto.ConnectionOpts,
	connID string) (wrappers.Wrap, error) {
	streamConn, err := p.tg.Dial(connID, opts)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot dial to Telegram")
	}
	streamConn = wrappers.NewCtx(ctx, cancel, streamConn)

	packetConn, err := p.tg.Init(opts, streamConn)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot handshake telegram")
	}

	return packetConn, nil
}

func (p *Proxy) middlePipe(src wrappers.WrapPacketReader, dst wrappers.WrapPacketWriter, wait *sync.WaitGroup, hacks *mtproto.Hacks) {
	defer wait.Done()

	for {
		hacks.SimpleAck = false
		hacks.QuickAck = false

		packet, err := src.Read()
		if err != nil {
			return
		}
		if _, err = dst.Write(packet); err != nil {
			return
		}
	}
}

func (p *Proxy) directPipe(src io.Reader, dst io.Writer, wait *sync.WaitGroup) {
	defer wait.Done()
	io.Copy(dst, src)

}

func NewProxy(conf *config.Config) *Proxy {
	var clientInit client.Init
	var tg telegram.Telegram

	if conf.UseMiddleProxy() {
		clientInit = client.MiddleInit
		tg = telegram.NewMiddleTelegram(conf)
	} else {
		clientInit = client.DirectInit
		tg = telegram.NewDirectTelegram(conf)
	}

	return &Proxy{
		conf:       conf,
		clientInit: clientInit,
		tg:         tg,
	}
}
