package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"slices"
	"strings"

	"github.com/9seconds/mtg/v2/antireplay"
	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/internal/desync"
	"github.com/9seconds/mtg/v2/internal/proxyprotocol"
	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/9seconds/mtg/v2/ipblocklist"
	"github.com/9seconds/mtg/v2/ipblocklist/files"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/network/v2"
	"github.com/9seconds/mtg/v2/stats"
	"github.com/pires/go-proxyproto"
	"github.com/rs/zerolog"
	"github.com/yl2chen/cidranger"
)

func makeLogger(conf *config.Config) mtglib.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.TimestampFieldName = "timestamp"
	zerolog.LevelFieldName = "level"

	if conf.Debug.Get(false) {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	baseLogger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	return logger.NewZeroLogger(baseLogger)
}

func makeNetwork(conf *config.Config, version string) (mtglib.Network, error) {
	resolver, err := network.GetDNS(conf.GetDNS())
	if err != nil {
		return nil, fmt.Errorf("cannot create DNS resolver: %w", err)
	}

	base := network.New(
		resolver,
		"",
		conf.Network.Timeout.TCP.Get(0),
		conf.Network.Timeout.HTTP.Get(0),
		conf.Network.Timeout.Idle.Get(0),
		net.KeepAliveConfig{
			Enable:   !conf.Network.KeepAlive.Disabled.Get(false),
			Idle:     conf.Network.KeepAlive.Idle.Get(0),
			Interval: conf.Network.KeepAlive.Interval.Get(0),
			Count:    int(conf.Network.KeepAlive.Count.Get(0)),
		},
		int(conf.Network.TCPNotSentLowat.Get(network.DefaultTCPNotSentLowat)),
	)

	proxyDialers := make([]mtglib.Network, len(conf.Network.Proxies))
	for idx, v := range conf.Network.Proxies {
		value, err := network.NewProxyNetwork(base, v.Get(nil))
		if err != nil {
			return nil, fmt.Errorf("cannot use %v for proxy url: %w", v.Get(nil), err)
		}
		proxyDialers[idx] = value
	}

	switch len(proxyDialers) {
	case 0:
		return base, nil
	case 1:
		return proxyDialers[0], nil
	}

	value, err := network.Join(proxyDialers...)
	if err != nil {
		panic(err)
	}

	return value, nil
}

func makeAntiReplayCache(conf *config.Config) mtglib.AntiReplayCache {
	if !conf.Defense.AntiReplay.Enabled.Get(false) {
		return antireplay.NewNoop()
	}

	return antireplay.NewStableBloomFilter(
		conf.Defense.AntiReplay.MaxSize.Get(antireplay.DefaultStableBloomFilterMaxSize),
		conf.Defense.AntiReplay.ErrorRate.Get(antireplay.DefaultStableBloomFilterErrorRate),
	)
}

func makeIPBlocklist(conf config.ListConfig,
	logger mtglib.Logger,
	ntw mtglib.Network,
	updateCallback ipblocklist.FireholUpdateCallback,
) (mtglib.IPBlocklist, error) {
	if !conf.Enabled.Get(false) {
		return ipblocklist.NewNoop(), nil
	}

	remoteURLs := []string{}
	localFiles := []string{}

	for _, v := range conf.URLs {
		if v.IsRemote() {
			remoteURLs = append(remoteURLs, v.String())
		} else {
			localFiles = append(localFiles, v.String())
		}
	}

	blocklist, err := ipblocklist.NewFirehol(logger.Named("ipblockist"),
		ntw,
		conf.DownloadConcurrency.Get(1),
		remoteURLs,
		localFiles,
		updateCallback)
	if err != nil {
		return nil, fmt.Errorf("incorrect parameters for firehol: %w", err)
	}

	go blocklist.Run(conf.UpdateEach.Get(ipblocklist.DefaultFireholUpdateEach))

	return blocklist, nil
}

func makeIPAllowlist(conf config.ListConfig,
	logger mtglib.Logger,
	ntw mtglib.Network,
	updateCallback ipblocklist.FireholUpdateCallback,
) (mtglib.IPBlocklist, error) {
	var (
		allowlist mtglib.IPBlocklist
		err       error
	)

	if !conf.Enabled.Get(false) {
		allowlist, err = ipblocklist.NewFireholFromFiles(
			logger.Named("ipblocklist"),
			1,
			[]files.File{
				files.NewMem([]*net.IPNet{
					cidranger.AllIPv4,
					cidranger.AllIPv6,
				}),
			},
			updateCallback,
		)

		go allowlist.Run(conf.UpdateEach.Get(ipblocklist.DefaultFireholUpdateEach))
	} else {
		allowlist, err = makeIPBlocklist(
			conf,
			logger,
			ntw,
			updateCallback,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("cannot build allowlist: %w", err)
	}

	return allowlist, nil
}

func makeEventStream(conf *config.Config, logger mtglib.Logger) (mtglib.EventStream, error) {
	factories := make([]events.ObserverFactory, 0, 2)

	if conf.Stats.StatsD.Enabled.Get(false) {
		statsdFactory, err := stats.NewStatsd(
			conf.Stats.StatsD.Address.Get(""),
			logger.Named("statsd"),
			conf.Stats.StatsD.MetricPrefix.Get(stats.DefaultStatsdMetricPrefix),
			conf.Stats.StatsD.TagFormat.Get(stats.DefaultStatsdTagFormat),
		)
		if err != nil {
			return nil, fmt.Errorf("cannot build statsd observer: %w", err)
		}

		factories = append(factories, statsdFactory.Make)
	}

	if conf.Stats.Prometheus.Enabled.Get(false) {
		prometheus := stats.NewPrometheus(
			conf.Stats.Prometheus.MetricPrefix.Get(stats.DefaultMetricPrefix),
			conf.Stats.Prometheus.HTTPPath.Get("/"),
		)

		listener, err := net.Listen("tcp", conf.Stats.Prometheus.BindTo.Get(""))
		if err != nil {
			return nil, fmt.Errorf("cannot start a listener for prometheus: %w", err)
		}

		go prometheus.Serve(listener) //nolint: errcheck

		factories = append(factories, prometheus.Make)
	}

	if len(factories) > 0 {
		return events.NewEventStream(factories), nil
	}

	return events.NewNoopStream(), nil
}

func warnSNIMismatch(conf *config.Config, ntw mtglib.Network, log mtglib.Logger) {
	host := conf.Secret.Host
	if host == "" {
		return
	}

	log = log.BindStr("hostname", host)

	res, err := runSNICheck(context.Background(), conf, net.DefaultResolver, ntw)
	if err != nil {
		log.WarningError("SNI-DNS check: cannot resolve secret hostname", err)
		return
	}

	if res.OurIP4 == "" && res.OurIP6 == "" {
		log.Warning("SNI-DNS check: cannot detect public IP address; set public-ipv4/public-ipv6 in config or run 'mtg doctor'")
		return
	}

	if len(res.ResolvedIP4) > 0 && !slices.Contains(res.ResolvedIP4, res.OurIP4) {
		log.
			BindStr("public_ip", res.OurIP4).
			BindStr("resolved", strings.Join(res.ResolvedIP4, ",")).
			Warning("SNI-DNS check: address mismatch")
	}

	if len(res.ResolvedIP6) > 0 && !slices.Contains(res.ResolvedIP6, res.OurIP6) {
		log.
			BindStr("public_ip", res.OurIP6).
			BindStr("resolved", strings.Join(res.ResolvedIP6, ",")).
			Warning("SNI-DNS check: address mismatch")
	}
}

func warnDeprecatedDomainFronting(conf *config.Config, log mtglib.Logger) {
	if conf.DomainFrontingIP.Value != nil {
		log.Warning(`config option "domain-fronting-ip" is deprecated and ignored; use "host" in [domain-fronting] instead`)
	}

	if conf.DomainFronting.IP.Value != nil {
		log.Warning(`config option "ip" in [domain-fronting] is deprecated and ignored; use "host" instead`)
	}
}

const dpiDesyncHandshakeWindowClamp = 256

func runProxy(conf *config.Config, version string) error { //nolint: funlen, cyclop
	logger := makeLogger(conf)

	logger.BindJSON("configuration", conf.String()).Debug("configuration")

	warnDeprecatedDomainFronting(conf, logger)

	eventStream, err := makeEventStream(conf, logger)
	if err != nil {
		return fmt.Errorf("cannot build event stream: %w", err)
	}

	ntw, err := makeNetwork(conf, version)
	if err != nil {
		return fmt.Errorf("cannot build network: %w", err)
	}

	warnSNIMismatch(conf, ntw, logger)

	blocklist, err := makeIPBlocklist(
		conf.Defense.Blocklist,
		logger.Named("blocklist"),
		ntw,
		func(ctx context.Context, size int) {
			eventStream.Send(ctx, mtglib.NewEventIPListSize(size, true))
		},
	)
	if err != nil {
		return fmt.Errorf("cannot build ip blocklist: %w", err)
	}

	allowlist, err := makeIPAllowlist(
		conf.Defense.Allowlist,
		logger.Named("allowlist"),
		ntw,
		func(ctx context.Context, size int) {
			eventStream.Send(ctx, mtglib.NewEventIPListSize(size, false))
		},
	)
	if err != nil {
		return fmt.Errorf("cannot build ip allowlist: %w", err)
	}

	windowClamp := 0
	if conf.DPIDesync.Get(false) {
		// Empirically chosen: small enough for Linux IPv4 DPI desync, but still
		// large enough for Telegram media after the post-handshake clamp restore.
		windowClamp = dpiDesyncHandshakeWindowClamp
	}

	doppelGangerURLs := make([]string, len(conf.Defense.Doppelganger.URLs))
	for i, v := range conf.Defense.Doppelganger.URLs {
		doppelGangerURLs[i] = v.String()
	}

	opts := mtglib.ProxyOpts{
		Logger:          logger,
		Network:         ntw,
		AntiReplayCache: makeAntiReplayCache(conf),
		IPBlocklist:     blocklist,
		IPAllowlist:     allowlist,
		EventStream:     eventStream,

		Secret:                      conf.Secret,
		Concurrency:                 conf.GetConcurrency(mtglib.DefaultConcurrency),
		DomainFrontingPort:          conf.GetDomainFrontingPort(mtglib.DefaultDomainFrontingPort),
		DomainFrontingHost:          conf.GetDomainFrontingHost(),
		DomainFrontingProxyProtocol: conf.GetDomainFrontingProxyProtocol(false),
		PreferIP:                    conf.PreferIP.Get(mtglib.DefaultPreferIP),
		AutoUpdate:                  conf.AutoUpdate.Get(false),

		AllowFallbackOnUnknownDC: conf.AllowFallbackOnUnknownDC.Get(false),
		TolerateTimeSkewness:     conf.TolerateTimeSkewness.Value,
		IdleTimeout:              conf.Network.Timeout.Idle.Get(mtglib.DefaultIdleTimeout),
		HandshakeTimeout:         conf.Network.Timeout.Handshake.Get(mtglib.DefaultHandshakeTimeout),

		DoppelGangerURLs:    doppelGangerURLs,
		DoppelGangerPerRaid: conf.Defense.Doppelganger.Repeats.Get(mtglib.DoppelGangerPerRaid),
		DoppelGangerEach:    conf.Defense.Doppelganger.UpdateEach.Get(mtglib.DoppelGangerEach),
		DoppelGangerDRS:     conf.Defense.Doppelganger.DRS.Get(false),

		DPIDesync: windowClamp > 0,
	}

	proxy, err := mtglib.NewProxy(opts)
	if err != nil {
		return fmt.Errorf("cannot create a proxy: %w", err)
	}

	listener, err := utils.NewListener(conf.BindTo.Get(""), windowClamp)
	if err != nil {
		return fmt.Errorf("cannot start proxy: %w", err)
	}

	if conf.ProxyProtocolListener.Get(false) {
		listener = &proxyprotocol.ListenerAdapter{
			Listener: proxyproto.Listener{
				Listener: listener,
			},
		}
	}

	ctx := utils.RootContext()

	if windowClamp > 0 {
		desyncSvc, err := desync.Start(int(conf.BindTo.Port))
		if err != nil {
			return fmt.Errorf("cannot start raw desync: %w", err)
		}
		defer desyncSvc.Close() //nolint: errcheck
	}

	go proxy.Serve(listener) //nolint: errcheck

	<-ctx.Done()
	listener.Close() //nolint: errcheck
	proxy.Shutdown()

	return nil
}
