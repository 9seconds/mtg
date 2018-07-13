package main

//go:generate scripts/generate_version.sh

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"syscall"
	"time"

	"github.com/juju/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/ntp"
	"github.com/9seconds/mtg/proxy"
	"github.com/9seconds/mtg/stats"
)

var (
	app = kingpin.New("mtg", "Simple MTPROTO proxy.")

	debug = app.Flag("debug", "Run in debug mode.").
		Short('d').
		Envar("MTG_DEBUG").
		Bool()
	verbose = app.Flag("verbose", "Run in verbose mode.").
		Short('v').
		Envar("MTG_VERBOSE").
		Bool()

	bindIP = app.Flag("bind-ip", "Which IP to bind to.").
		Short('b').
		Envar("MTG_IP").
		Default("127.0.0.1").
		IP()
	bindPort = app.Flag("bind-port", "Which port to bind to.").
			Short('p').
			Envar("MTG_PORT").
			Default("3128").
			Uint16()

	publicIPv4 = app.Flag("public-ipv4", "Which IPv4 address is public.").
			Short('4').
			Envar("MTG_IPV4").
			IP()
	publicIPv4Port = app.Flag("public-ipv4-port", "Which IPv4 port is public. Default is 'bind-port' value.").
			Envar("MTG_IPV4_PORT").
			Uint16()

	publicIPv6 = app.Flag("public-ipv6", "Which IPv6 address is public.").
			Short('6').
			Envar("MTG_IPV6").
			IP()
	publicIPv6Port = app.Flag("public-ipv6-port", "Which IPv6 port is public. Default is 'bind-port' value.").
			Envar("MTG_IPV6_PORT").
			Uint16()

	statsIP = app.Flag("stats-ip", "Which IP bind stats server to.").
		Short('t').
		Envar("MTG_STATS_IP").
		Default("127.0.0.1").
		IP()
	statsPort = app.Flag("stats-port", "Which port bind stats to.").
			Short('q').
			Envar("MTG_STATS_PORT").
			Default("3129").
			Uint16()

	statsdIP = app.Flag("statsd-ip", "Which IP should we use for working with statsd.").
			Envar("MTG_STATSD_IP").
			String()
	statsdPort = app.Flag("statsd-port", "Which port should we use for working with statsd.").
			Envar("MTG_STATSD_PORT").
			Default("8125").
			Uint16()
	statsdNetwork = app.Flag("statsd-network", "Which network is used to work with statsd. Only 'tcp' and 'udp' are supported.").
			Envar("MTG_STATSD_NETWORK").
			Default("udp").
			String()
	statsdPrefix = app.Flag("statsd-prefix", "Which bucket prefix should we use for sending stats to statsd.").
			Envar("MTG_STATSD_PREFIX").
			Default("mtg").
			String()
	statsdTagsFormat = app.Flag("statsd-tags-format", "Which tag format should we use to send stats metrics. Valid options are 'datadog' and 'influxdb'.").
				Envar("MTG_STATSD_TAGS_FORMAT").
				String()
	statsdTags = app.Flag("statsd-tags", "Tags to use for working with statsd (specified as 'key=value').").
			Envar("MTG_STATSD_TAGS").
			StringMap()

	secret = app.Arg("secret", "Secret of this proxy.").Required().String()
	adtag  = app.Arg("adtag", "ADTag of the proxy.").String()
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	app.Version(version)

}

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	err := setRLimit()
	if err != nil {
		usage(err.Error())
	}

	conf, err := config.NewConfig(*debug, *verbose,
		*bindIP, *bindPort,
		*publicIPv4, *publicIPv4Port,
		*publicIPv6, *publicIPv6Port,
		*statsIP, *statsPort,
		*secret, *adtag,
		*statsdIP, *statsdPort, *statsdNetwork, *statsdPrefix,
		*statsdTagsFormat, *statsdTags,
	)
	if err != nil {
		usage(err.Error())
	}

	atom := zap.NewAtomicLevel()
	if conf.Debug {
		atom.SetLevel(zapcore.DebugLevel)
	} else if conf.Verbose {
		atom.SetLevel(zapcore.InfoLevel)
	} else {
		atom.SetLevel(zapcore.ErrorLevel)
	}
	encoderCfg := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stderr),
		atom,
	))
	zap.ReplaceGlobals(logger)
	defer logger.Sync() // nolint: errcheck

	printURLs(conf.GetURLs())

	if conf.UseMiddleProxy() {
		zap.S().Infow("Use middle proxy connection to Telegram")
		if diff, err := ntp.Fetch(); err != nil {
			zap.S().Warnw("Could not fetch time data from NTP")
		} else {
			if diff >= time.Second {
				usage(fmt.Sprintf("You choose to use middle proxy but your clock drift (%s) is bigger than 1 second. Please, sync your time", diff))
			}
			go ntp.AutoUpdate()
		}
	} else {
		zap.S().Infow("Use direct connection to Telegram")
	}

	if err := stats.Start(conf); err != nil {
		panic(err)
	}

	server := proxy.NewProxy(conf)
	if err := server.Serve(); err != nil {
		zap.S().Fatalw("Server stopped", "error", err)
	}
}

func setRLimit() (err error) {
	rLimit := syscall.Rlimit{}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		err = errors.Annotate(err, "Cannot get rlimit")
		return
	}
	rLimit.Cur = rLimit.Max

	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		err = errors.Annotate(err, "Cannot set rlimit")
	}

	return
}

func printURLs(data interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(data)
	if err != nil {
		panic(err)
	}
}

func usage(msg string) {
	io.WriteString(os.Stderr, msg+"\n") // nolint: errcheck
	os.Exit(1)
}
