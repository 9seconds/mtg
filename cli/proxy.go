package cli

import (
	"fmt"
	"net"
	"os"

	"github.com/9seconds/mtg/v2/antireplay"
	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/ipblocklist"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/stats"
	"github.com/9seconds/mtg/v2/timeattack"
	"github.com/9seconds/mtg/v2/utils"
	"github.com/rs/zerolog"
)

type Proxy struct {
	base
}

func (c *Proxy) Run(cli *CLI, version string) error {
	if err := c.ReadConfig(version); err != nil {
		return fmt.Errorf("cannot init config: %w", err)
	}

	return c.Execute()
}

func (c *Proxy) Execute() error {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.TimestampFieldName = "timestamp"
	zerolog.LevelFieldName = "level"

	if c.Config.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	ctx := utils.RootContext()
	opts := mtglib.ProxyOpts{
		Logger:             logger.NewZeroLogger(zerolog.New(os.Stdout).With().Timestamp().Logger()),
		Network:            c.Network,
		AntiReplayCache:    antireplay.NewNoop(),
		IPBlocklist:        ipblocklist.NewNoop(),
		TimeAttackDetector: timeattack.NewNoop(),
		EventStream:        events.NewNoopStream(),

		Secret:             c.Config.Secret,
		BufferSize:         c.Config.TCPBuffer.Value(mtglib.DefaultBufferSize),
		DomainFrontingPort: c.Config.DomainFrontingPort.Value(mtglib.DefaultDomainFrontingPort),
		IdleTimeout:        c.Config.Network.Timeout.Idle.Value(mtglib.DefaultIdleTimeout),
		PreferIP:           c.Config.PreferIP.Value(mtglib.DefaultPreferIP),
	}

	opts.Logger.BindStr("configuration", c.Config.String()).Debug("configuration")

	c.setupAntiReplayCache(&opts)
	c.setupTimeAttackDetector(&opts)

	if err := c.setupIPBlocklist(&opts); err != nil {
		return fmt.Errorf("cannot setup ipblocklist: %w", err)
	}

	if err := c.setupEventStream(&opts); err != nil {
		return fmt.Errorf("cannot setup event stream: %w", err)
	}

	proxy, err := mtglib.NewProxy(opts)
	if err != nil {
		return fmt.Errorf("cannot create a proxy: %w", err)
	}

	listener, err := net.Listen("tcp", c.Config.BindTo.String())
	if err != nil {
		return fmt.Errorf("cannot start proxy: %w", err)
	}

	go proxy.Serve(listener) // nolint: errcheck

	<-ctx.Done()
	listener.Close()
	proxy.Shutdown()

	return nil
}

func (c *Proxy) setupAntiReplayCache(opts *mtglib.ProxyOpts) {
	if !c.Config.Defense.AntiReplay.Enabled {
		return
	}

	opts.AntiReplayCache = antireplay.NewStableBloomFilter(
		c.Config.Defense.AntiReplay.MaxSize.Value(antireplay.DefaultMaxSize),
		c.Config.Defense.AntiReplay.ErrorRate.Value(antireplay.DefaultErrorRate),
	)
}

func (c *Proxy) setupTimeAttackDetector(opts *mtglib.ProxyOpts) {
	if !c.Config.Defense.Time.Enabled {
		return
	}

	opts.TimeAttackDetector = timeattack.NewDetector(
		c.Config.Defense.Time.AllowSkewness.Value(timeattack.DefaultDuration),
	)
}

func (c *Proxy) setupIPBlocklist(opts *mtglib.ProxyOpts) error {
	if !c.Config.Defense.Blocklist.Enabled {
		return nil
	}

	remoteURLs := []string{}
	localFiles := []string{}

	for _, v := range c.Config.Defense.Blocklist.URLs {
		if v.IsRemote() {
			remoteURLs = append(remoteURLs, v.String())
		} else {
			localFiles = append(localFiles, v.String())
		}
	}

	firehol, err := ipblocklist.NewFirehol(opts.Logger.Named("ipblockist"),
		c.Network,
		c.Config.Defense.Blocklist.DownloadConcurrency,
		remoteURLs,
		localFiles)
	if err != nil {
		return err // nolint: wrapcheck
	}

	go firehol.Run(c.Config.Defense.Blocklist.UpdateEach.Value(ipblocklist.DefaultUpdateEach))

	opts.IPBlocklist = firehol

	return nil
}

func (c *Proxy) setupEventStream(opts *mtglib.ProxyOpts) error {
	factories := make([]events.ObserverFactory, 0, 2)

	if c.Config.Stats.StatsD.Enabled {
		statsdFactory, err := stats.NewStatsd(
			c.Config.Stats.StatsD.Address.String(),
			opts.Logger.Named("statsd"),
			c.Config.Stats.StatsD.MetricPrefix.Value(stats.DefaultStatsdMetricPrefix),
			c.Config.Stats.StatsD.TagFormat.Value(stats.DefaultStatsdTagFormat))
		if err != nil {
			return fmt.Errorf("cannot build statsd observer: %w", err)
		}

		factories = append(factories, statsdFactory.Make)
	}

	if c.Config.Stats.Prometheus.Enabled {
		prometheus := stats.NewPrometheus(
			c.Config.Stats.Prometheus.MetricPrefix.Value(stats.DefaultMetricPrefix),
			c.Config.Stats.Prometheus.HTTPPath.Value("/"),
		)

		listener, err := net.Listen("tcp", c.Config.Stats.Prometheus.BindTo.String())
		if err != nil {
			return fmt.Errorf("cannot start a listener for prometheus: %w", err)
		}

		go prometheus.Serve(listener) // nolint: errcheck

		factories = append(factories, prometheus.Make)
	}

	if len(factories) > 0 {
		opts.EventStream = events.NewEventStream(factories)
	}

	return nil
}
