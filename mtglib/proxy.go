package mtglib

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/internal/faketls"
	"github.com/9seconds/mtg/v2/mtglib/internal/faketls/record"
	"github.com/9seconds/mtg/v2/mtglib/internal/obfuscated2"
	"github.com/9seconds/mtg/v2/mtglib/internal/relay"
	"github.com/9seconds/mtg/v2/mtglib/internal/telegram"
	"github.com/panjf2000/ants/v2"
)

type Proxy struct {
	ctx             context.Context
	ctxCancel       context.CancelFunc
	streamWaitGroup sync.WaitGroup

	idleTimeout        time.Duration
	bufferSize         int
	domainFrontingPort int
	workerPool         *ants.PoolWithFunc
	telegram           *telegram.Telegram

	secret             Secret
	network            Network
	antiReplayCache    AntiReplayCache
	timeAttackDetector TimeAttackDetector
	ipBlocklist        IPBlocklist
	eventStream        EventStream
	logger             Logger
}

func (p *Proxy) DomainFrontingAddress() string {
	return net.JoinHostPort(p.secret.Host, strconv.Itoa(p.domainFrontingPort))
}

func (p *Proxy) ServeConn(conn net.Conn) {
	p.streamWaitGroup.Add(1)
	defer p.streamWaitGroup.Done()

	ctx := newStreamContext(p.ctx, p.logger, conn)
	defer ctx.Close()

	go func() {
		<-ctx.Done()
		ctx.Close()
	}()

	p.eventStream.Send(ctx, NewEventStart(ctx.streamID, ctx.ClientIP()))
	ctx.logger.Info("Stream has been started")

	defer func() {
		p.eventStream.Send(ctx, NewEventFinish(ctx.streamID))
		ctx.logger.Info("Stream has been finished")
	}()

	if !p.doFakeTLSHandshake(ctx) {
		return
	}

	if err := p.doObfuscated2Handshake(ctx); err != nil {
		p.logger.InfoError("obfuscated2 handshake is failed", err)

		return
	}

	if err := p.doTelegramCall(ctx); err != nil {
		p.logger.WarningError("cannot dial to telegram", err)

		return
	}

	rel := relay.AcquireRelay(ctx,
		p.logger.Named("relay"), p.bufferSize, p.idleTimeout)
	defer relay.ReleaseRelay(rel)

	if err := rel.Process(ctx.clientConn, ctx.telegramConn); err != nil {
		p.logger.DebugError("relay has been finished", err)
	}
}

func (p *Proxy) Serve(listener net.Listener) error {
	p.streamWaitGroup.Add(1)
	defer p.streamWaitGroup.Done()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("cannot accept a new connection: %w", err)
		}

		ipAddr := conn.RemoteAddr().(*net.TCPAddr).IP
		logger := p.logger.BindStr("ip", ipAddr.String())

		if p.ipBlocklist.Contains(ipAddr) {
			conn.Close()
			logger.Info("ip was blacklisted")
			p.eventStream.Send(p.ctx, NewEventIPBlocklisted(ipAddr))

			continue
		}

		err = p.workerPool.Invoke(conn)

		switch {
		case err == nil:
		case errors.Is(err, ants.ErrPoolClosed):
			return nil
		case errors.Is(err, ants.ErrPoolOverload):
			logger.Info("connection was concurrency limited")
			p.eventStream.Send(p.ctx, NewEventConcurrencyLimited())
		}

		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		default:
		}
	}
}

func (p *Proxy) Shutdown() {
	p.ctxCancel()
	p.streamWaitGroup.Wait()
	p.workerPool.Release()
}

func (p *Proxy) doFakeTLSHandshake(ctx *streamContext) bool {
	rec := record.AcquireRecord()
	defer record.ReleaseRecord(rec)

	rewind := newConnRewind(ctx.clientConn)

	if err := rec.Read(rewind); err != nil {
		p.logger.InfoError("cannot read client hello", err)
		p.doDomainFronting(ctx, rewind)

		return false
	}

	hello, err := faketls.ParseClientHello(p.secret.Key[:], rec.Payload.Bytes())
	if err != nil {
		p.logger.InfoError("cannot parse client hello", err)
		p.doDomainFronting(ctx, rewind)

		return false
	}

	if hello.Host != "" && hello.Host != p.secret.Host {
		p.logger.BindStr("hostname", hello.Host).Info("incorrect domain was found in SNI")
		p.doDomainFronting(ctx, rewind)

		return false
	}

	if err := p.timeAttackDetector.Valid(hello.Time); err != nil {
		p.logger.InfoError("invalid faketls time", err)
		p.doDomainFronting(ctx, rewind)

		return false
	}

	if p.antiReplayCache.SeenBefore(hello.SessionID) {
		p.logger.Warning("replay attack has been detected!")
		p.eventStream.Send(p.ctx, NewEventReplayAttack(ctx.streamID))
		p.doDomainFronting(ctx, rewind)

		return false
	}

	if err := faketls.SendWelcomePacket(rewind, p.secret.Key[:], hello); err != nil {
		p.logger.InfoError("cannot send welcome packet", err)

		return false
	}

	ctx.clientConn = &faketls.Conn{
		Conn: ctx.clientConn,
	}

	return true
}

func (p *Proxy) doObfuscated2Handshake(ctx *streamContext) error {
	dc, encryptor, decryptor, err := obfuscated2.ClientHandshake(p.secret.Key[:], ctx.clientConn)
	if err != nil {
		return fmt.Errorf("cannot process client handshake: %w", err)
	}

	ctx.dc = dc
	ctx.logger = ctx.logger.BindInt("dc", dc)
	ctx.clientConn = obfuscated2.Conn{
		Conn:      ctx.clientConn,
		Encryptor: encryptor,
		Decryptor: decryptor,
	}

	return nil
}

func (p *Proxy) doTelegramCall(ctx *streamContext) error {
	conn, err := p.telegram.Dial(ctx, ctx.dc)
	if err != nil {
		return fmt.Errorf("cannot dial to Telegram: %w", err)
	}

	encryptor, decryptor, err := obfuscated2.ServerHandshake(conn)
	if err != nil {
		conn.Close()

		return fmt.Errorf("cannot perform obfuscated2 handshake: %w", err)
	}

	ctx.telegramConn = obfuscated2.Conn{
		Conn: connTraffic{
			Conn:     conn,
			streamID: ctx.streamID,
			stream:   p.eventStream,
			ctx:      ctx,
		},
		Encryptor: encryptor,
		Decryptor: decryptor,
	}

	p.eventStream.Send(ctx,
		NewEventConnectedToDC(ctx.streamID, conn.RemoteAddr().(*net.TCPAddr).IP, ctx.dc))

	return nil
}

func (p *Proxy) doDomainFronting(ctx *streamContext, conn *connRewind) {
	p.eventStream.Send(p.ctx, NewEventDomainFronting(ctx.streamID))
	conn.Rewind()

	frontConn, err := p.network.DialContext(ctx, "tcp", p.DomainFrontingAddress())
	if err != nil {
		p.logger.WarningError("cannot dial to the fronting domain", err)

		return
	}

	frontConn = connTraffic{
		Conn:     frontConn,
		ctx:      ctx,
		streamID: ctx.streamID,
		stream:   p.eventStream,
	}

	rel := relay.AcquireRelay(ctx,
		p.logger.Named("domain-fronting"), p.bufferSize, p.idleTimeout)
	defer relay.ReleaseRelay(rel)

	if err := rel.Process(conn, frontConn); err != nil {
		p.logger.DebugError("domain fronting relay has been finished", err)
	}
}

func NewProxy(opts ProxyOpts) (*Proxy, error) {
	if err := opts.valid(); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	tg, err := telegram.New(opts.Network, opts.getPreferIP())
	if err != nil {
		return nil, fmt.Errorf("cannot build telegram dialer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	proxy := &Proxy{
		ctx:                ctx,
		ctxCancel:          cancel,
		secret:             opts.Secret,
		network:            opts.Network,
		antiReplayCache:    opts.AntiReplayCache,
		timeAttackDetector: opts.TimeAttackDetector,
		ipBlocklist:        opts.IPBlocklist,
		eventStream:        opts.EventStream,
		logger:             opts.getLogger("proxy"),
		domainFrontingPort: opts.getDomainFrontingPort(),
		idleTimeout:        opts.getIdleTimeout(),
		bufferSize:         opts.getBufferSize(),
		telegram:           tg,
	}

	pool, err := ants.NewPoolWithFunc(opts.getConcurrency(),
		func(arg interface{}) {
			proxy.ServeConn(arg.(net.Conn))
		},
		ants.WithLogger(opts.getLogger("ants")),
		ants.WithNonblocking(true))
	if err != nil {
		panic(err)
	}

	proxy.workerPool = pool

	return proxy, nil
}
