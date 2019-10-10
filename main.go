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
	generateSecretType = generateSecretCommand.Arg("type",
		"A type of secret to generate. Valid options are 'simple', 'secured' and 'tls'").
		Required().
		Enum("simple", "secured", "tls")

	proxyCommand = app.Command("proxy",
		"Run new proxy instance")
	proxyDebug = proxyCommand.Flag("debug",
		"Run in debug mode.").
		Short('d').
		Envar("MTG_DEBUG").
		Bool()
	proxyVerbose = proxyCommand.Flag("verbose",
		"Run in verbose mode.").
		Short('v').
		Envar("MTG_VERBOSE").
		Bool()
	proxyBind = proxyCommand.Flag("bind",
		"Host:Port to bind proxy to.").
		Short('b').
		Envar("MTG_BIND").
		Default("0.0.0.0:3128").
		TCP()
	proxyPublicIPv4 = proxyCommand.Flag("public-ipv4",
		"Which IPv4 host:port to use.").
		Short('4').
		Envar("MTG_IPV4").
		TCP()
	proxyPublicIPv6 = proxyCommand.Flag("public-ipv6",
		"Which IPv6 host:port to use.").
		Short('6').
		Envar("MTG_IPV6").
		TCP()
	proxyStatsBind = proxyCommand.Flag("stats-bind",
		"Which Host:Port to bind stats server to.").
		Short('t').
		Envar("MTG_STATS_BIND").
		Default("127.0.0.1:3129").
		TCP()
	proxyStatsNamespace = proxyCommand.Flag("stats-namespace",
		"Which namespace to use for Prometheus.").
		Envar("MTG_STATS_NAMESPACE").
		Default("mtg").
		String()
	proxyStatsdAddress = proxyCommand.Flag("statsd-addr",
		"Host:port of statsd server").
		Envar("MTG_STATSD_ADDR").
		TCP()
	proxyStatsdNetwork = proxyCommand.Flag("statsd-network",
		"Which network is used to work with statsd. Only 'tcp' and 'udp' are supported.").
		Envar("MTG_STATSD_NETWORK").
		Default("udp").
		Enum("udp", "tcp")
	proxyStatsdTagsFormat = proxyCommand.Flag("statsd-tags-format",
		"Which tag format should we use to send stats metrics. Valid options are 'datadog' and 'influxdb'.").
		Envar("MTG_STATSD_TAGS_FORMAT").
		Default("influxdb").
		Enum("datadog", "influxdb")
	proxyStatsdTags = proxyCommand.Flag("statsd-tags",
		"Tags to use for working with statsd (specified as 'key=value').").
		Envar("MTG_STATSD_TAGS").
		StringMap()
	proxyWriteBufferSize = proxyCommand.Flag("write-buffer",
		"Write buffer size in bytes. You can think about it as a buffer from client to Telegram.").
		Short('w').
		Envar("MTG_BUFFER_WRITE").
		Default("65536KB").
		Bytes()
	proxyReadBufferSize = proxyCommand.Flag("read-buffer",
		"Read buffer size in bytes. You can think about it as a buffer from Telegram to client.").
		Short('r').
		Envar("MTG_BUFFER_READ").
		Default("131072KB").
		Bytes()
	proxyAntiReplayMaxSize = proxyCommand.Flag("anti-replay-max-size",
		"Max size of antireplay cache in megabytes.").
		Envar("MTG_ANTIREPLAY_MAXSIZE").
		Default("128").
		Int()
	proxyAntiReplayEvictionTime = proxyCommand.Flag("anti-replay-eviction-time",
		"Eviction time period for obfuscated2 handshakes").
		Envar("MTG_ANTIREPLAY_EVICTIONTIME").
		Default("168h").
		Duration()
	proxySecret = proxyCommand.Arg("secret", "Secret of this proxy.").Required().HexBytes()
	proxyAdtag  = proxyCommand.Arg("adtag", "ADTag of the proxy.").HexBytes()
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
		cli.Generate(*generateSecretType)

	case proxyCommand.FullCommand():
		err := config.Init(
			config.Opt{Option: config.OptionTypeDebug, Value: *proxyDebug},
			config.Opt{Option: config.OptionTypeVerbose, Value: *proxyVerbose},
			config.Opt{Option: config.OptionTypeBind, Value: *proxyBind},
			config.Opt{Option: config.OptionTypePublicIPv4, Value: *proxyPublicIPv4},
			config.Opt{Option: config.OptionTypePublicIPv6, Value: *proxyPublicIPv6},
			config.Opt{Option: config.OptionTypeStatsBind, Value: *proxyStatsBind},
			config.Opt{Option: config.OptionTypeStatsNamespace, Value: *proxyStatsNamespace},
			config.Opt{Option: config.OptionTypeStatsdAddress, Value: *proxyStatsdAddress},
			config.Opt{Option: config.OptionTypeStatsdNetwork, Value: *proxyStatsdNetwork},
			config.Opt{Option: config.OptionTypeStatsdTagsFormat, Value: *proxyStatsdTagsFormat},
			config.Opt{Option: config.OptionTypeStatsdTags, Value: *proxyStatsdTags},
			config.Opt{Option: config.OptionTypeWriteBufferSize, Value: *proxyWriteBufferSize},
			config.Opt{Option: config.OptionTypeReadBufferSize, Value: *proxyReadBufferSize},
			config.Opt{Option: config.OptionTypeAntiReplayMaxSize, Value: *proxyAntiReplayMaxSize},
			config.Opt{Option: config.OptionTypeAntiReplayEvictionTime, Value: *proxyAntiReplayEvictionTime},
			config.Opt{Option: config.OptionTypeSecret, Value: *proxySecret},
			config.Opt{Option: config.OptionTypeAdtag, Value: *proxyAdtag},
		)
		if err != nil {
			cli.Fatal(err)
		}

		if err := cli.Proxy(); err != nil {
			cli.Fatal(err)
		}
	}
}
