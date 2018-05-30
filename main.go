package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	readTimeout = app.Flag("read-timeout", "Socket read timeout").
			Short('r').
			Envar("MTG_READ_TIMEOUT").
			Default("30s").
			Duration()
	writeTimeout = app.Flag("write-timeout", "Socket write timeout").
			Short('w').
			Envar("MTG_WRITE_TIMEOUT").
			Default("30s").
			Duration()
	serverName = app.Flag("server-name",
		"Which server name to use. Default is IP address resolved by ipify.").
		Short('s').
		Envar("MTG_SERVER").
		String()

	secret = app.Arg("secret", "Secret of this proxy.").String()
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	secretBytes, err := hex.DecodeString(*secret)
	if err != nil {
		usage("Secret has to be hexadecimal string.")
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

	printURLs()
	srv := proxy.NewServer(*bindIP, int(*bindPort), secretBytes, logger,
		*readTimeout, *writeTimeout)
	if err := srv.Serve(); err != nil {
		logger.Fatal(err.Error())
	}
}

func usage(msg string) {
	io.WriteString(os.Stderr, msg+"\n")
	os.Exit(1)
}

func printURLs() {
	values := url.Values{}
	values.Set("server", *serverName)
	values.Set("port", strconv.Itoa(int(*bindPort)))
	values.Set("secret", *secret)

	tgURL := url.URL{
		Scheme:   "tg",
		Host:     "proxy",
		RawQuery: values.Encode(),
	}
	fmt.Println(tgURL.String())

	tgURL.Scheme = "https"
	tgURL.Host = "t.me"
	tgURL.Path = "proxy"
	fmt.Println(tgURL.String())
}
