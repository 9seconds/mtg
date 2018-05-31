package main

//go:generate scripts/generate_version.sh

import (
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/9seconds/mtg/proxy"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
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
		Short('i').
		Envar("MTG_IP").
		Default("127.0.0.1").
		IP()
	bindPort = app.Flag("bind-port", "Which port to bind to.").
			Short('p').
			Envar("MTG_PORT").
			Default("3128").
			Uint16()
	portToShow = app.Flag("show-bind-port",
		"Which port to show in URL. Default is the value of bind-port").
		Short('a').
		Envar("MTG_SHOW_PORT").
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
	readTimeout = app.Flag("read-timeout", "Socket read timeout.").
			Short('r').
			Envar("MTG_READ_TIMEOUT").
			Default("30s").
			Duration()
	writeTimeout = app.Flag("write-timeout", "Socket write timeout.").
			Short('w').
			Envar("MTG_WRITE_TIMEOUT").
			Default("30s").
			Duration()
	serverName = app.Flag("server-name",
		"Which server name to use. Default is IP address resolved by ipify.").
		Short('s').
		Envar("MTG_SERVER").
		String()
	preferIPv6 = app.Flag("prefer-ipv6", "Use IPv6").
			Short('6').
			Envar("MTG_USE_IPV6").
			Bool()

	secret = app.Arg("secret", "Secret of this proxy.").String()
)

func main() {
	app.Version(version)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	secretBytes, err := hex.DecodeString(*secret)
	if err != nil {
		usage("Secret has to be hexadecimal string.")
	}

	if *portToShow == 0 {
		*portToShow = *bindPort
	}

	if *serverName == "" {
		resp, err := http.Get("https://api.ipify.org")
		if err != nil || resp.StatusCode != http.StatusOK {
			usage("Cannot get local IP address.")
		}
		myIPBytes, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			usage("Cannot get local IP address.")
		}
		*serverName = strings.TrimSpace(string(myIPBytes))
	}

	atom := zap.NewAtomicLevel()
	if *debug {
		atom.SetLevel(zapcore.DebugLevel)
	} else if *verbose {
		atom.SetLevel(zapcore.InfoLevel)
	} else {
		atom.SetLevel(zapcore.ErrorLevel)
	}
	encoderCfg := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stderr),
		atom,
	)).Sugar()

	stat := proxy.NewStats(*serverName, *portToShow, *secret)
	go stat.Serve(*statsIP, *statsPort)

	srv := proxy.NewServer(*bindIP, int(*bindPort), secretBytes, logger,
		*readTimeout, *writeTimeout, *preferIPv6, stat)
	if err := srv.Serve(); err != nil {
		logger.Fatal(err.Error())
	}
}

func usage(msg string) {
	io.WriteString(os.Stderr, msg+"\n")
	os.Exit(1)
}
