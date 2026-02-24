package mtglib

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/internal/dc"
	"github.com/9seconds/mtg/v2/mtglib/internal/faketls"
	"github.com/9seconds/mtg/v2/mtglib/internal/faketls/record"
	"github.com/9seconds/mtg/v2/mtglib/internal/obfuscation"
	"github.com/9seconds/mtg/v2/mtglib/internal/relay"
	"github.com/panjf2000/ants/v2"
)

// Proxy is an MTPROTO proxy structure.
type Proxy struct {
	ctx             context.Context
	ctxCancel       context.CancelFunc
	streamWaitGroup sync.WaitGroup

	allowFallbackOnUnknownDC bool
	tolerateTimeSkewness     time.Duration
	domainFrontingPort       int
	workerPool               *ants.PoolWithFunc
	telegram                 *dc.Telegram
	clientObfuscatror        obfuscation.Obfuscator

	secret          Secret
	network         Network
	antiReplayCache AntiReplayCache
	blocklist       IPBlocklist
	allowlist       IPBlocklist
	eventStream     EventStream
	logger          Logger
}

// DomainFrontingAddress returns a host:port pair for a fronting domain.
func (p *Proxy) DomainFrontingAddress() string {
	return net.JoinHostPort(p.secret.Host, strconv.Itoa(p.domainFrontingPort))
}

// ServeConn serves a connection. We do not check IP blocklist and concurrency
// limit here.
func (p *Proxy) ServeConn(conn essentials.Conn) {
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

	if err := p.doObfuscatedHandshake(ctx); err != nil {
		p.logger.InfoError("obfuscated handshake is failed", err)

		return
	}

	if err := p.doTelegramCall(ctx); err != nil {
		p.logger.WarningError("cannot dial to telegram", err)

		return
	}

	relay.Relay(
		ctx,
		ctx.logger.Named("relay"),
		ctx.telegramConn,
		ctx.clientConn,
	)
}

// Serve starts a proxy on a given listener.
func (p *Proxy) Serve(listener net.Listener) error {
	p.streamWaitGroup.Add(1)
	defer p.streamWaitGroup.Done()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-p.ctx.Done():
				return nil
			default:
				return fmt.Errorf("cannot accept a new connection: %w", err)
			}
		}

		ipAddr := conn.RemoteAddr().(*net.TCPAddr).IP //nolint: forcetypeassert
		logger := p.logger.BindStr("ip", ipAddr.String())

		if !p.allowlist.Contains(ipAddr) {
			conn.Close() //nolint: errcheck
			logger.Info("ip was rejected by allowlist")
			p.eventStream.Send(p.ctx, NewEventIPAllowlisted(ipAddr))

			continue
		}

		if p.blocklist.Contains(ipAddr) {
			conn.Close() //nolint: errcheck
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
	}
}

// Shutdown 'gracefully' shutdowns all connections. Please remember that it
// does not close an underlying listener.
func (p *Proxy) Shutdown() {
	p.ctxCancel()
	p.streamWaitGroup.Wait()
	p.workerPool.Release()

	p.allowlist.Shutdown()
	p.blocklist.Shutdown()
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

	if err := hello.Valid(p.secret.Host, p.tolerateTimeSkewness); err != nil {
		p.logger.
			BindStr("hostname", hello.Host).
			BindStr("hello-time", hello.Time.String()).
			InfoError("invalid faketls client hello", err)
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

func (p *Proxy) doObfuscatedHandshake(ctx *streamContext) error {
	dc, conn, err := p.clientObfuscatror.ReadHandshake(ctx.clientConn)
	if err != nil {
		return fmt.Errorf("cannot process client handshake: %w", err)
	}

	ctx.dc = dc
	ctx.clientConn = conn
	ctx.logger = ctx.logger.BindInt("dc", dc)

	return nil
}

func (p *Proxy) doTelegramCall(ctx *streamContext) error {
	dcid := ctx.dc

	addresses := p.telegram.GetAddresses(dcid)
	if len(addresses) == 0 && p.allowFallbackOnUnknownDC {
		ctx.logger = ctx.logger.BindInt("original_dc", dcid)
		ctx.logger.Warning("unknown DC, fallbacks")
		ctx.dc = dc.DefaultDC
		addresses = p.telegram.GetAddresses(dc.DefaultDC)
	}

	var (
		conn      essentials.Conn
		err       error
		foundAddr dc.Addr
	)

	for _, addr := range addresses {
		conn, err = p.network.Dial(addr.Network, addr.Address)
		if err == nil {
			foundAddr = addr
			break
		}
	}
	if err != nil {
		return fmt.Errorf("no addresses to call: %w", err)
	}

	tgConn, err := foundAddr.Obfuscator.SendHandshake(conn, ctx.dc)
	if err != nil {
		conn.Close() // nolint: errcheck
		return fmt.Errorf("cannot perform server handshake: %w", err)
	}

	ctx.telegramConn = connTraffic{
		Conn:     tgConn,
		streamID: ctx.streamID,
		stream:   p.eventStream,
		ctx:      ctx,
	}

	p.eventStream.Send(ctx,
		NewEventConnectedToDC(ctx.streamID,
			conn.RemoteAddr().(*net.TCPAddr).IP, //nolint: forcetypeassert
			ctx.dc),
	)

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

	relay.Relay(
		ctx,
		ctx.logger.Named("domain-fronting"),
		frontConn,
		conn,
	)
}

// NewProxy makes a new proxy instance.
func NewProxy(opts ProxyOpts) (*Proxy, error) {
	if err := opts.valid(); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	tg, err := dc.New(opts.getPreferIP())
	if err != nil {
		return nil, fmt.Errorf("cannot build telegram dc fetcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	logger := opts.getLogger("proxy")
	updatersLogger := logger.Named("telegram-updaters")

	proxy := &Proxy{
		ctx:                      ctx,
		ctxCancel:                cancel,
		secret:                   opts.Secret,
		network:                  opts.Network,
		antiReplayCache:          opts.AntiReplayCache,
		blocklist:                opts.IPBlocklist,
		allowlist:                opts.IPAllowlist,
		eventStream:              opts.EventStream,
		logger:                   logger,
		domainFrontingPort:       opts.getDomainFrontingPort(),
		tolerateTimeSkewness:     opts.getTolerateTimeSkewness(),
		allowFallbackOnUnknownDC: opts.AllowFallbackOnUnknownDC,
		telegram:                 tg,
		clientObfuscatror: obfuscation.Obfuscator{
			Secret: opts.Secret.Key[:],
		},
	}

	publicConfigUpdater := dc.NewPublicConfigUpdater(
		tg,
		updatersLogger.Named("public-config"),
		opts.Network.MakeHTTPClient(nil),
	)
	go publicConfigUpdater.Run(ctx, dc.PublicConfigUpdateURLv4, "tcp4")
	go publicConfigUpdater.Run(ctx, dc.PublicConfigUpdateURLv6, "tcp6")

	pool, err := ants.NewPoolWithFunc(opts.getConcurrency(),
		func(arg any) {
			proxy.ServeConn(arg.(essentials.Conn)) //nolint: forcetypeassert
		},
		ants.WithLogger(opts.getLogger("ants")),
		ants.WithNonblocking(true))
	if err != nil {
		panic(err)
	}

	proxy.workerPool = pool

	return proxy, nil
}
