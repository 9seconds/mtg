package mtglib

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/internal/faketls/clienthello"
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

	idleTimeout time.Duration
	bufferSize  int
	workerPool  *ants.PoolWithFunc
	telegram    *telegram.Telegram

	secret             Secret
	antiReplayCache    AntiReplayCache
	timeAttackDetector TimeAttackDetector
	ipBlocklist        IPBlocklist
	eventStream        EventStream
	logger             Logger
}

func (p *Proxy) ServeConn(conn net.Conn) {
	ctx := newStreamContext(p.ctx, p.logger, conn)
	defer ctx.Close()

	go func() {
		<-ctx.Done()
		ctx.Close()
	}()

	p.eventStream.Send(ctx, EventStart{
		CreatedAt: time.Now(),
		ConnID:    ctx.connID,
		RemoteIP:  ctx.ClientIP(),
	})
	ctx.logger.Info("Stream has been started")

	defer func() {
		p.eventStream.Send(ctx, EventFinish{
			CreatedAt: time.Now(),
			ConnID:    ctx.connID,
		})
		ctx.logger.Info("Stream has been finished")
	}()

	if err := p.doFakeTLSHandshake(ctx); err != nil {
		p.logger.InfoError("faketls handshake is failed", err)

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

	rel := relay.AcquireRelay(ctx, p.logger.Named("relay"), p.bufferSize, p.idleTimeout)
	defer relay.ReleaseRelay(rel)

	if err := rel.Process(ctx.clientConn, ctx.telegramConn); err != nil {
		p.logger.DebugError("relay has been finished", err)
	}
}

func (p *Proxy) Serve(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("cannot accept a new connection: %w", err)
		}

		if addr := conn.RemoteAddr().(*net.TCPAddr).IP; p.ipBlocklist.Contains(addr) {
			conn.Close()
			p.eventStream.Send(p.ctx, EventIPBlocklisted{
				CreatedAt: time.Now(),
				RemoteIP:  addr,
			})

			continue
		}

		err = p.workerPool.Invoke(conn)

		switch {
		case err == nil:
		case errors.Is(err, ants.ErrPoolClosed):
			return nil
		case errors.Is(err, ants.ErrPoolOverload):
			p.eventStream.Send(p.ctx, EventConcurrencyLimited{
				CreatedAt: time.Now(),
			})
		}
	}
}

func (p *Proxy) Shutdown() {
	p.ctxCancel()
	p.streamWaitGroup.Wait()
	p.workerPool.Release()
}

func (p *Proxy) doFakeTLSHandshake(ctx *streamContext) error {
	clientHelloRecord := record.AcquireRecord()
	defer record.ReleaseRecord(clientHelloRecord)

	if err := clientHelloRecord.Read(ctx.clientConn); err != nil {
		return fmt.Errorf("cannot read client hello: %w", err)
	}

	hello, _ := clienthello.ParseHandshake(p.secret.Key[:],
		clientHelloRecord.Payload.Bytes())
	fmt.Println(hello)

	return fmt.Errorf("SUCCESS")
}

func (p *Proxy) doObfuscated2Handshake(ctx *streamContext) error {
	dc, encryptor, decryptor, err := obfuscated2.ClientHandshake(p.secret.Key[:], ctx.clientConn)
	if err != nil {
		return fmt.Errorf("cannot process client handshake: %w", err)
	}

	ctx.dc = dc
	ctx.logger = ctx.logger.BindInt("dc", dc)
	ctx.clientConn = &obfuscated2.Conn{
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

	ctx.telegramConn = &obfuscated2.Conn{
		Conn: connTelegramTraffic{
			Conn:   conn,
			connID: ctx.connID,
			stream: p.eventStream,
			ctx:    ctx,
		},
		Encryptor: encryptor,
		Decryptor: decryptor,
	}

	p.eventStream.Send(ctx, EventConnectedToDC{
		CreatedAt: time.Now(),
		ConnID:    ctx.connID,
		RemoteIP:  conn.RemoteAddr().(*net.TCPAddr).IP,
		DC:        ctx.dc,
	})

	return nil
}

func NewProxy(opts ProxyOpts) (*Proxy, error) { // nolint: cyclop
	switch {
	case opts.Network == nil:
		return nil, ErrNetworkIsNotDefined
	case opts.AntiReplayCache == nil:
		return nil, ErrAntiReplayCacheIsNotDefined
	case opts.IPBlocklist == nil:
		return nil, ErrIPBlocklistIsNotDefined
	case opts.EventStream == nil:
		return nil, ErrEventStreamIsNotDefined
	case opts.TimeAttackDetector == nil:
		return nil, ErrTimeAttackDetectorIsNotDefined
	case opts.Logger == nil:
		return nil, ErrLoggerIsNotDefined
	case !opts.Secret.Valid():
		return nil, ErrSecretInvalid
	}

	tg, err := telegram.New(opts.Network, opts.PreferIP)
	if err != nil {
		return nil, fmt.Errorf("cannot build telegram dialer: %w", err)
	}

	concurrency := opts.Concurrency
	if concurrency == 0 {
		concurrency = DefaultConcurrency
	}

	idleTimeout := opts.IdleTimeout
	if idleTimeout < 1 {
		idleTimeout = DefaultIdleTimeout
	}

	bufferSize := opts.BufferSize
	if bufferSize < 1 {
		bufferSize = DefaultBufferSize
	}

	ctx, cancel := context.WithCancel(context.Background())
	proxy := &Proxy{
		ctx:                ctx,
		ctxCancel:          cancel,
		secret:             opts.Secret,
		antiReplayCache:    opts.AntiReplayCache,
		timeAttackDetector: opts.TimeAttackDetector,
		ipBlocklist:        opts.IPBlocklist,
		eventStream:        opts.EventStream,
		logger:             opts.Logger.Named("proxy"),
		idleTimeout:        idleTimeout,
		bufferSize:         int(bufferSize),
		telegram:           tg,
	}

	pool, err := ants.NewPoolWithFunc(int(concurrency), func(arg interface{}) {
		proxy.ServeConn(arg.(net.Conn))
	}, ants.WithLogger(opts.Logger.Named("ants")))
	if err != nil {
		return nil, fmt.Errorf("cannot initialize a pool: %w", err)
	}

	proxy.workerPool = pool

	return proxy, nil
}
