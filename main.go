package main

//go:generate scripts/generate_version.sh

import (
	"encoding/json"
	"io"
	"math/rand"
	"os"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/proxy"
	"github.com/juju/errors"
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

	statsIP = app.Flag("stats-ip", "Which IP bind stats server to").
		Short('t').
		Envar("MTG_STATS_IP").
		Default("127.0.0.1").
		IP()
	statsPort = app.Flag("stats-port", "Which port bind stats to.").
			Short('q').
			Envar("MTG_STATS_PORT").
			Default("3129").
			Uint16()

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
	defer logger.Sync()

	var server *proxy.Proxy
	if len(conf.AdTag) == 0 {
		server = proxy.NewProxyDirect(conf)
	} else {
		server = proxy.NewProxyMiddle(conf)
	}

	printURLs(conf.GetURLs())

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
