package mtglib

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/internal/dc"
	"github.com/9seconds/mtg/v2/mtglib/internal/doppel"
	"github.com/9seconds/mtg/v2/mtglib/internal/obfuscation"
	"github.com/9seconds/mtg/v2/mtglib/internal/relay"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls/fake"
	"github.com/panjf2000/ants/v2"
)

// Proxy is an MTPROTO proxy structure.
type Proxy struct {
	ctx             context.Context
	ctxCancel       context.CancelFunc
	streamWaitGroup sync.WaitGroup

	allowFallbackOnUnknownDC    bool
	tolerateTimeSkewness        time.Duration
	idleTimeout                 time.Duration
	domainFrontingPort          int
	domainFrontingIP            string
	domainFrontingProxyProtocol bool
	workerPool                  *ants.PoolWithFunc
	telegram                    *dc.Telegram
	configUpdater               *dc.PublicConfigUpdater
	doppelGanger                *doppel.Ganger

	stats       *ProxyStats
	secrets     []Secret
	secretNames []string
	network         Network
	antiReplayCache AntiReplayCache
	blocklist       IPBlocklist
	allowlist       IPBlocklist
	eventStream     EventStream
	logger          Logger
}

// DomainFrontingAddress returns a host:port pair for a fronting domain.
// If DomainFrontingIP is set, it is used instead of resolving the hostname.
func (p *Proxy) DomainFrontingAddress() string {
	// All secrets share the same host (enforced by validation),
	// so we use the first one.
	host := p.secrets[0].Host
	if p.domainFrontingIP != "" {
		host = p.domainFrontingIP
	}

	return net.JoinHostPort(host, strconv.Itoa(p.domainFrontingPort))
}

// ServeConn serves a connection. We do not check IP blocklist and concurrency
// limit here.
func (p *Proxy) ServeConn(conn essentials.Conn) {
	p.streamWaitGroup.Add(1)
	defer p.streamWaitGroup.Done()

	ctx := newStreamContext(p.ctx, p.logger, conn)
	defer ctx.Close()

	stop := context.AfterFunc(ctx, func() {
		ctx.Close()
	})
	defer stop()

	p.eventStream.Send(ctx, NewEventStart(ctx.streamID, ctx.ClientIP()))
	ctx.logger.Info("Stream has been started")

	defer func() {
		p.eventStream.Send(ctx, NewEventFinish(ctx.streamID))
		ctx.logger.Info("Stream has been finished")
	}()

	if !p.doFakeTLSHandshake(ctx) {
		return
	}

	p.stats.OnConnect(ctx.secretName)
	p.stats.UpdateLastSeen(ctx.secretName)

	defer p.stats.OnDisconnect(ctx.secretName)

	clientConn, err := p.doppelGanger.NewConn(ctx.clientConn)
	if err != nil {
		ctx.logger.InfoError("cannot wrap into doppelganger connection", err)
		return
	}
	defer clientConn.Stop()

	ctx.clientConn = clientConn

	if err := p.doObfuscatedHandshake(ctx); err != nil {
		ctx.logger.InfoError("obfuscated handshake is failed", err)
		return
	}

	if err := p.doTelegramCall(ctx); err != nil {
		ctx.logger.WarningError("cannot dial to telegram", err)
		return
	}

	countedClientConn := newCountingConn(ctx.clientConn, p.stats, ctx.secretName)

	relay.Relay(
		ctx,
		ctx.logger.Named("relay"),
		ctx.telegramConn,
		countedClientConn,
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
			conn.Close() //nolint: errcheck
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
	p.configUpdater.Wait()
	p.doppelGanger.Shutdown()

	p.allowlist.Shutdown()
	p.blocklist.Shutdown()
}

func (p *Proxy) doFakeTLSHandshake(ctx *streamContext) bool {
	rewind := newConnRewind(ctx.clientConn)

	// Build a slice of secret keys to try during HMAC validation.
	secretKeys := make([][]byte, len(p.secrets))
	for i := range p.secrets {
		secretKeys[i] = p.secrets[i].Key[:]
	}

	result, err := fake.ReadClientHelloMulti(
		rewind,
		secretKeys,
		p.secrets[0].Host,
		p.tolerateTimeSkewness,
	)
	if err != nil {
		p.logger.InfoError("cannot read client hello", err)
		p.doDomainFronting(ctx, rewind)
		return false
	}

	if p.antiReplayCache.SeenBefore(result.Hello.SessionID) {
		p.logger.Warning("replay attack has been detected!")
		p.eventStream.Send(p.ctx, NewEventReplayAttack(ctx.streamID))
		p.doDomainFronting(ctx, rewind)
		return false
	}

	matchedSecret := p.secrets[result.MatchedIndex]
	ctx.matchedSecretKey = matchedSecret.Key[:]
	ctx.secretName = p.secretNames[result.MatchedIndex]
	ctx.logger = ctx.logger.BindStr("secret_name", ctx.secretName)

	gangerNoise := p.doppelGanger.NoiseParams()
	noiseParams := fake.NoiseParams{Mean: gangerNoise.Mean, Jitter: gangerNoise.Jitter}

	if err := fake.SendServerHello(ctx.clientConn, matchedSecret.Key[:], result.Hello, noiseParams); err != nil {
		p.logger.InfoError("cannot send welcome packet", err)
		return false
	}

	ctx.clientConn = tls.New(ctx.clientConn, true, false)

	return true
}

func (p *Proxy) doObfuscatedHandshake(ctx *streamContext) error {
	// Use the secret key that was matched during the FakeTLS handshake.
	obfs := obfuscation.Obfuscator{
		Secret: ctx.matchedSecretKey,
	}

	dc, conn, err := obfs.ReadHandshake(ctx.clientConn)
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
	if conn == nil {
		return fmt.Errorf("no available addresses for DC %d", ctx.dc)
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

	telegramHost, _, err := net.SplitHostPort(foundAddr.Address)
	if err != nil {
		conn.Close() //nolint: errcheck

		return fmt.Errorf("cannot parse telegram address %s: %w", foundAddr.Address, err)
	}

	p.eventStream.Send(ctx,
		NewEventConnectedToDC(ctx.streamID,
			net.ParseIP(telegramHost),
			ctx.dc),
	)

	return nil
}

func (p *Proxy) doDomainFronting(ctx *streamContext, conn *connRewind) {
	p.eventStream.Send(p.ctx, NewEventDomainFronting(ctx.streamID))
	conn.Rewind()

	nativeDialer := p.network.NativeDialer()
	fConn, err := nativeDialer.DialContext(ctx, "tcp", p.DomainFrontingAddress())
	if err != nil {
		p.logger.WarningError("cannot dial to the fronting domain", err)

		return
	}

	frontConn := essentials.WrapNetConn(fConn)

	if p.domainFrontingProxyProtocol {
		frontConn = newConnProxyProtocol(ctx.clientConn, frontConn)
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
		connIdleTimeout{Conn: frontConn, timeout: p.idleTimeout},
		connIdleTimeout{Conn: conn, timeout: p.idleTimeout},
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

	secretsMap := opts.getSecrets()
	secretNames := make([]string, 0, len(secretsMap))

	for name := range secretsMap {
		secretNames = append(secretNames, name)
	}

	sort.Strings(secretNames)

	secretsList := make([]Secret, 0, len(secretsMap))

	for _, name := range secretNames {
		secretsList = append(secretsList, secretsMap[name])
	}

	stats := NewProxyStats()
	for _, name := range secretNames {
		stats.PreRegister(name)
	}

	if opts.APIBindTo != "" {
		stats.StartServer(ctx, opts.APIBindTo, logger)
	}

	proxy := &Proxy{
		ctx:                      ctx,
		ctxCancel:                cancel,
		stats:                    stats,
		secrets:                  secretsList,
		secretNames:              secretNames,
		network:                  opts.Network,
		antiReplayCache:          opts.AntiReplayCache,
		blocklist:                opts.IPBlocklist,
		allowlist:                opts.IPAllowlist,
		eventStream:              opts.EventStream,
		logger:                   logger,
		domainFrontingPort:       opts.getDomainFrontingPort(),
		domainFrontingIP:         opts.DomainFrontingIP,
		tolerateTimeSkewness:     opts.getTolerateTimeSkewness(),
		idleTimeout:              opts.getIdleTimeout(),
		allowFallbackOnUnknownDC: opts.AllowFallbackOnUnknownDC,
		telegram:                 tg,
		doppelGanger: doppel.NewGanger(
			ctx,
			opts.Network,
			logger.Named("doppelganger"),
			opts.DoppelGangerEach,
			int(opts.DoppelGangerPerRaid),
			opts.DoppelGangerURLs,
			opts.DoppelGangerDRS,
		),
		configUpdater: dc.NewPublicConfigUpdater(
			tg,
			updatersLogger.Named("public-config"),
			opts.Network.MakeHTTPClient(nil),
		),
		domainFrontingProxyProtocol: opts.DomainFrontingProxyProtocol,
	}

	proxy.doppelGanger.Run()

	if opts.AutoUpdate {
		proxy.configUpdater.Run(ctx, dc.PublicConfigUpdateURLv4, "tcp4")
		proxy.configUpdater.Run(ctx, dc.PublicConfigUpdateURLv6, "tcp6")
	}

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
