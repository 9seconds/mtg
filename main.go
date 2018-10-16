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

	debug = app.Flag("debug",
		"Run in debug mode.").
		Short('d').
		Envar("MTG_DEBUG").
		Bool()
	verbose = app.Flag("verbose",
		"Run in verbose mode.").
		Short('v').
		Envar("MTG_VERBOSE").
		Bool()

	bindIP = app.Flag("bind-ip",
		"Which IP to bind to.").
		Short('b').
		Envar("MTG_IP").
		Default("127.0.0.1").
		IP()
	bindPort = app.Flag("bind-port",
		"Which port to bind to.").
		Short('p').
		Envar("MTG_PORT").
		Default("3128").
		Uint16()

	publicIPv4 = app.Flag("public-ipv4",
		"Which IPv4 address is public.").
		Short('4').
		Envar("MTG_IPV4").
		IP()
	publicIPv4Port = app.Flag("public-ipv4-port",
		"Which IPv4 port is public. Default is 'bind-port' value.").
		Envar("MTG_IPV4_PORT").
		Uint16()

	publicIPv6 = app.Flag("public-ipv6",
		"Which IPv6 address is public.").
		Short('6').
		Envar("MTG_IPV6").
		IP()
	publicIPv6Port = app.Flag("public-ipv6-port",
		"Which IPv6 port is public. Default is 'bind-port' value.").
		Envar("MTG_IPV6_PORT").
		Uint16()

	statsIP = app.Flag("stats-ip",
		"Which IP bind stats server to.").
		Short('t').
		Envar("MTG_STATS_IP").
		Default("127.0.0.1").
		IP()
	statsPort = app.Flag("stats-port",
		"Which port bind stats to.").
		Short('q').
		Envar("MTG_STATS_PORT").
		Default("3129").
		Uint16()

	statsdIP = app.Flag("statsd-ip",
		"Which IP should we use for working with statsd.").
		Envar("MTG_STATSD_IP").
		String()
	statsdPort = app.Flag("statsd-port",
		"Which port should we use for working with statsd.").
		Envar("MTG_STATSD_PORT").
		Default("8125").
		Uint16()
	statsdNetwork = app.Flag("statsd-network",
		"Which network is used to work with statsd. Only 'tcp' and 'udp' are supported.").
		Envar("MTG_STATSD_NETWORK").
		Default("udp").
		String()
	statsdPrefix = app.Flag("statsd-prefix",
		"Which bucket prefix should we use for sending stats to statsd.").
		Envar("MTG_STATSD_PREFIX").
		Default("mtg").
		String()
	statsdTagsFormat = app.Flag("statsd-tags-format",
		"Which tag format should we use to send stats metrics. Valid options are 'datadog' and 'influxdb'.").
		Envar("MTG_STATSD_TAGS_FORMAT").
		String()
	statsdTags = app.Flag("statsd-tags",
		"Tags to use for working with statsd (specified as 'key=value').").
		Envar("MTG_STATSD_TAGS").
		StringMap()

	writeBufferSize = app.Flag("write-buffer",
		"Write buffer size in bytes. You can think about it as a buffer from client to Telegram.").
		Short('w').
		Envar("MTG_BUFFER_WRITE").
		Default("65536").
		Uint32()
	readBufferSize = app.Flag("read-buffer",
		"Read buffer size in bytes. You can think about it as a buffer from Telegram to client.").
		Short('r').
		Envar("MTG_BUFFER_READ").
		Default("131072").
		Uint32()
	secureOnly = app.Flag("secure-only",
		"Support clients with dd-secrets only.").
		Short('s').
		Envar("MTG_SECURE_ONLY").
		Bool()

	secret = app.Arg("secret", "Secret of this proxy.").Required().HexBytes()
	adtag  = app.Arg("adtag", "ADTag of the proxy.").HexBytes()
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	app.Version(version)
	app.HelpFlag.Short('h')
}

func main() { // nolint: gocyclo
	kingpin.MustParse(app.Parse(os.Args[1:]))

	err := setRLimit()
	if err != nil {
		usage(err.Error())
	}

	conf, err := config.NewConfig(*debug, *verbose,
		*writeBufferSize, *readBufferSize,
		*bindIP, *publicIPv4, *publicIPv6, *statsIP,
		*bindPort, *publicIPv4Port, *publicIPv6Port, *statsPort, *statsdPort,
		*statsdIP, *statsdNetwork, *statsdPrefix, *statsdTagsFormat,
		*statsdTags, *secureOnly,
		*secret, *adtag,
	)
	if err != nil {
		usage(err.Error())
	}

	atom := zap.NewAtomicLevel()
	switch {
	case conf.Debug:
		atom.SetLevel(zapcore.DebugLevel)
	case conf.Verbose:
		atom.SetLevel(zapcore.InfoLevel)
	default:
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
				usage(fmt.Sprintf("You choose to use middle proxy but your clock drift (%s) "+
					"is bigger than 1 second. Please, sync your time", diff))
			}
			go ntp.AutoUpdate()
		}
	} else {
		zap.S().Infow("Use direct connection to Telegram")
	}

	if err := stats.Init(conf); err != nil {
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
	io.WriteString(os.Stderr, msg+"\n") // nolint: errcheck, gosec
	os.Exit(1)
}
