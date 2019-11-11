package main

import (
	"math/rand"
	"os"
	"time"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/9seconds/mtg/cli"
	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/utils"
)

var version = "dev" // this has to be set by build ld flags

var (
	app = kingpin.New("mtg", "Simple MTPROTO proxy.")

	generateSecretCommand = app.Command("generate-secret",
		"Generate new secret")
	generateCloakHost = generateSecretCommand.Flag("cloak-host",
		"A host to use for TLS cloaking.").
		Short('c').
		Default("storage.googleapis.com").
		String()
	generateSecretType = generateSecretCommand.Arg("type",
		"A type of secret to generate. Valid options are 'simple', 'secured' and 'tls'").
		Required().
		Enum("simple", "secured", "tls")

	runCommand = app.Command("run",
		"Run new proxy instance")
	runDebug = runCommand.Flag("debug",
		"Run in debug mode.").
		Short('d').
		Envar("MTG_DEBUG").
		Bool()
	runVerbose = runCommand.Flag("verbose",
		"Run in verbose mode.").
		Short('v').
		Envar("MTG_VERBOSE").
		Bool()
	runBind = runCommand.Flag("bind",
		"Host:Port to bind proxy to.").
		Short('b').
		Envar("MTG_BIND").
		Default("0.0.0.0:3128").
		TCP()
	runPublicIPv4 = runCommand.Flag("public-ipv4",
		"Which IPv4 host:port to use.").
		Short('4').
		Envar("MTG_IPV4").
		TCP()
	runPublicIPv6 = runCommand.Flag("public-ipv6",
		"Which IPv6 host:port to use.").
		Short('6').
		Envar("MTG_IPV6").
		TCP()
	runStatsBind = runCommand.Flag("stats-bind",
		"Which Host:Port to bind stats server to.").
		Short('t').
		Envar("MTG_STATS_BIND").
		Default("127.0.0.1:3129").
		TCP()
	runStatsNamespace = runCommand.Flag("stats-namespace",
		"Which namespace to use for Prometheus.").
		Envar("MTG_STATS_NAMESPACE").
		Default("mtg").
		String()
	runStatsdAddress = runCommand.Flag("statsd-addr",
		"Host:port of statsd server").
		Envar("MTG_STATSD_ADDR").
		TCP()
	runStatsdNetwork = runCommand.Flag("statsd-network",
		"Which network is used to work with statsd. Only 'tcp' and 'udp' are supported.").
		Envar("MTG_STATSD_NETWORK").
		Default("udp").
		Enum("udp", "tcp")
	runStatsdTagsFormat = runCommand.Flag("statsd-tags-format",
		"Which tag format should we use to send stats metrics. Valid options are 'datadog' and 'influxdb'.").
		Envar("MTG_STATSD_TAGS_FORMAT").
		Default("influxdb").
		Enum("datadog", "influxdb")
	runStatsdTags = runCommand.Flag("statsd-tags",
		"Tags to use for working with statsd (specified as 'key=value').").
		Envar("MTG_STATSD_TAGS").
		StringMap()
	runWriteBufferSize = runCommand.Flag("write-buffer",
		"Write buffer size in bytes. You can think about it as a buffer from client to Telegram.").
		Short('w').
		Envar("MTG_BUFFER_WRITE").
		Default("65536KB").
		Bytes()
	runReadBufferSize = runCommand.Flag("read-buffer",
		"Read buffer size in bytes. You can think about it as a buffer from Telegram to client.").
		Short('r').
		Envar("MTG_BUFFER_READ").
		Default("131072KB").
		Bytes()
	runTLSCloakPort = runCommand.Flag("cloak-port",
		"Port which should be used for host cloaking.").
		Envar("MTG_CLOAK_PORT").
		Default("443").
		Uint16()
	runAntiReplayMaxSize = runCommand.Flag("anti-replay-max-size",
		"Max size of antireplay cache.").
		Envar("MTG_ANTIREPLAY_MAXSIZE").
		Default("128MB").
		Bytes()
	runMultiplexPerConnection = runCommand.Flag("multiplex-per-connection",
		"How many clients can share a single connection to Telegram.").
		Envar("MTG_MULTIPLEX_PERCONNECTION").
		Default("50").
		Uint()
	runSecret = runCommand.Arg("secret", "Secret of this proxy.").Required().HexBytes()
	runAdtag  = runCommand.Arg("adtag", "ADTag of the proxy.").HexBytes()
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	app.Version(version)
	app.HelpFlag.Short('h')

	if err := utils.SetLimits(); err != nil {
		cli.Fatal(err)
	}

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case generateSecretCommand.FullCommand():
		cli.Generate(*generateSecretType, *generateCloakHost)
	case runCommand.FullCommand():
		err := config.Init(
			config.Opt{Option: config.OptionTypeDebug, Value: *runDebug},
			config.Opt{Option: config.OptionTypeVerbose, Value: *runVerbose},
			config.Opt{Option: config.OptionTypeBind, Value: *runBind},
			config.Opt{Option: config.OptionTypePublicIPv4, Value: *runPublicIPv4},
			config.Opt{Option: config.OptionTypePublicIPv6, Value: *runPublicIPv6},
			config.Opt{Option: config.OptionTypeStatsBind, Value: *runStatsBind},
			config.Opt{Option: config.OptionTypeStatsNamespace, Value: *runStatsNamespace},
			config.Opt{Option: config.OptionTypeStatsdAddress, Value: *runStatsdAddress},
			config.Opt{Option: config.OptionTypeStatsdNetwork, Value: *runStatsdNetwork},
			config.Opt{Option: config.OptionTypeStatsdTagsFormat, Value: *runStatsdTagsFormat},
			config.Opt{Option: config.OptionTypeStatsdTags, Value: *runStatsdTags},
			config.Opt{Option: config.OptionTypeWriteBufferSize, Value: *runWriteBufferSize},
			config.Opt{Option: config.OptionTypeReadBufferSize, Value: *runReadBufferSize},
			config.Opt{Option: config.OptionTypeCloakPort, Value: *runTLSCloakPort},
			config.Opt{Option: config.OptionTypeAntiReplayMaxSize, Value: *runAntiReplayMaxSize},
			config.Opt{Option: config.OptionTypeMultiplexPerConnection, Value: *runMultiplexPerConnection},
			config.Opt{Option: config.OptionTypeSecret, Value: *runSecret},
			config.Opt{Option: config.OptionTypeAdtag, Value: *runAdtag},
		)
		if err != nil {
			cli.Fatal(err)
		}

		if err := cli.Proxy(); err != nil {
			cli.Fatal(err)
		}
	}
}
