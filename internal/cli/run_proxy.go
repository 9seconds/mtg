package cli

import (
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/9seconds/mtg/v2/antireplay"
	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/9seconds/mtg/v2/ipblocklist"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/network"
	"github.com/9seconds/mtg/v2/stats"
	"github.com/rs/zerolog"
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
	tcpTimeout := conf.Network.Timeout.TCP.Get(network.DefaultTimeout)
	httpTimeout := conf.Network.Timeout.HTTP.Get(network.DefaultHTTPTimeout)
	dohIP := conf.Network.DOHIP.Get(net.ParseIP(network.DefaultDOHHostname)).String()
	bufferSize := conf.TCPBuffer.Get(network.DefaultBufferSize)
	userAgent := "mtg/" + version

	baseDialer, err := network.NewDefaultDialer(tcpTimeout, int(bufferSize))
	if err != nil {
		return nil, fmt.Errorf("cannot build a default dialer: %w", err)
	}

	if len(conf.Network.Proxies) == 0 {
		return network.NewNetwork(baseDialer, userAgent, dohIP, httpTimeout) // nolint: wrapcheck
	}

	proxyURLs := make([]*url.URL, 0, len(conf.Network.Proxies))

	for _, v := range conf.Network.Proxies {
		if value := v.Get(nil); value != nil {
			proxyURLs = append(proxyURLs, value)
		}
	}

	if len(proxyURLs) == 1 {
		socksDialer, err := network.NewSocks5Dialer(baseDialer, proxyURLs[0])
		if err != nil {
			return nil, fmt.Errorf("cannot build socks5 dialer: %w", err)
		}

		return network.NewNetwork(socksDialer, userAgent, dohIP, httpTimeout) // nolint: wrapcheck
	}

	socksDialer, err := network.NewLoadBalancedSocks5Dialer(baseDialer, proxyURLs)
	if err != nil {
		return nil, fmt.Errorf("cannot build socks5 dialer: %w", err)
	}

	return network.NewNetwork(socksDialer, userAgent, dohIP, httpTimeout) // nolint: wrapcheck
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

func makeIPBlocklist(conf config.ListConfig, logger mtglib.Logger, ntw mtglib.Network) (mtglib.IPBlocklist, error) {
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

	firehol, err := ipblocklist.NewFirehol(logger.Named("ipblockist"),
		ntw,
		conf.DownloadConcurrency.Get(1),
		remoteURLs,
		localFiles)
	if err != nil {
		return nil, fmt.Errorf("incorrect parameters for firehol: %w", err)
	}

	return firehol, nil
}

func makeEventStream(conf *config.Config, logger mtglib.Logger) (mtglib.EventStream, error) {
	factories := make([]events.ObserverFactory, 0, 2) // nolint: gomnd

	if conf.Stats.StatsD.Enabled.Get(false) {
		statsdFactory, err := stats.NewStatsd(
			conf.Stats.StatsD.Address.Get(""),
			logger.Named("statsd"),
			conf.Stats.StatsD.MetricPrefix.Get(stats.DefaultStatsdMetricPrefix),
			conf.Stats.StatsD.TagFormat.Get(stats.DefaultStatsdTagFormat))
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

		go prometheus.Serve(listener) // nolint: errcheck

		factories = append(factories, prometheus.Make)
	}

	if len(factories) > 0 {
		return events.NewEventStream(factories), nil
	}

	return events.NewNoopStream(), nil
}

func runProxy(conf *config.Config, version string) error { // nolint: funlen
	logger := makeLogger(conf)

	logger.BindJSON("configuration", conf.String()).Debug("configuration")

	ntw, err := makeNetwork(conf, version)
	if err != nil {
		return fmt.Errorf("cannot build network: %w", err)
	}

	blocklist, err := makeIPBlocklist(conf.Defense.Blocklist, logger, ntw)
	if err != nil {
		return fmt.Errorf("cannot build ip blocklist: %w", err)
	}

	var whitelist mtglib.IPBlocklist

	if conf.Defense.Allowlist.Enabled.Get(false) {
		whlist, err := makeIPBlocklist(conf.Defense.Allowlist, logger, ntw)
		if err != nil {
			return fmt.Errorf("cannot build ip blocklist: %w", err)
		}

		whitelist = whlist
	}

	eventStream, err := makeEventStream(conf, logger)
	if err != nil {
		return fmt.Errorf("cannot build event stream: %w", err)
	}

	opts := mtglib.ProxyOpts{
		Logger:          logger,
		Network:         ntw,
		AntiReplayCache: makeAntiReplayCache(conf),
		IPBlocklist:     blocklist,
		IPWhitelist:     whitelist,
		EventStream:     eventStream,

		Secret:             conf.Secret,
		BufferSize:         conf.TCPBuffer.Get(mtglib.DefaultBufferSize),
		DomainFrontingPort: conf.DomainFrontingPort.Get(mtglib.DefaultDomainFrontingPort),
		PreferIP:           conf.PreferIP.Get(mtglib.DefaultPreferIP),

		AllowFallbackOnUnknownDC: conf.AllowFallbackOnUnknownDC.Get(false),
		TolerateTimeSkewness:     conf.TolerateTimeSkewness.Value,
	}

	proxy, err := mtglib.NewProxy(opts)
	if err != nil {
		return fmt.Errorf("cannot create a proxy: %w", err)
	}

	listener, err := utils.NewListener(conf.BindTo.Get(""), int(opts.BufferSize))
	if err != nil {
		return fmt.Errorf("cannot start proxy: %w", err)
	}

	ctx := utils.RootContext()

	go proxy.Serve(listener) // nolint: errcheck

	<-ctx.Done()
	listener.Close()
	proxy.Shutdown()

	return nil
}
