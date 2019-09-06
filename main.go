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
	proxyBindIP = proxyCommand.Flag("bind-ip",
		"Which IP to bind to.").
		Short('b').
		Envar("MTG_IP").
		Default("127.0.0.1").
		IP()
	proxyBindPort = proxyCommand.Flag("bind-port",
		"Which port to bind to.").
		Short('p').
		Envar("MTG_PORT").
		Default("3128").
		Uint16()
	proxyPublicIPv4 = proxyCommand.Flag("public-ipv4",
		"Which IPv4 address is public.").
		Short('4').
		Envar("MTG_IPV4").
		IP()
	proxyPublicIPv4Port = proxyCommand.Flag("public-ipv4-port",
		"Which IPv4 port is public. Default is 'bind-port' value.").
		Envar("MTG_IPV4_PORT").
		Uint16()
	proxyPublicIPv6 = proxyCommand.Flag("public-ipv6",
		"Which IPv6 address is public.").
		Short('6').
		Envar("MTG_IPV6").
		IP()
	proxyPublicIPv6Port = proxyCommand.Flag("public-ipv6-port",
		"Which IPv6 port is public. Default is 'bind-port' value.").
		Envar("MTG_IPV6_PORT").
		Uint16()
	proxyStatsIP = proxyCommand.Flag("stats-ip",
		"Which IP bind stats server to.").
		Short('t').
		Envar("MTG_STATS_IP").
		Default("127.0.0.1").
		IP()
	proxyStatsPort = proxyCommand.Flag("stats-port",
		"Which port bind stats to.").
		Short('q').
		Envar("MTG_STATS_PORT").
		Default("3129").
		Uint16()
	proxyStatsdIP = proxyCommand.Flag("statsd-ip",
		"Which IP should we use for working with statsd.").
		Envar("MTG_STATSD_IP").
		IP()
	proxyStatsdPort = proxyCommand.Flag("statsd-port",
		"Which port should we use for working with statsd.").
		Envar("MTG_STATSD_PORT").
		Default("8125").
		Uint16()
	proxyStatsdNetwork = proxyCommand.Flag("statsd-network",
		"Which network is used to work with statsd. Only 'tcp' and 'udp' are supported.").
		Envar("MTG_STATSD_NETWORK").
		Default("udp").
		Enum("udp", "tcp")
	proxyStatsdPrefix = proxyCommand.Flag("statsd-prefix",
		"Which bucket prefix should we use for sending stats to statsd.").
		Envar("MTG_STATSD_PREFIX").
		Default("mtg").
		String()
	proxyStatsdTagsFormat = proxyCommand.Flag("statsd-tags-format",
		"Which tag format should we use to send stats metrics. Valid options are 'datadog' and 'influxdb'.").
		Envar("MTG_STATSD_TAGS_FORMAT").
		Default("influxdb").
		Enum("datadog", "influxdb")
	proxyStatsdTags = proxyCommand.Flag("statsd-tags",
		"Tags to use for working with statsd (specified as 'key=value').").
		Envar("MTG_STATSD_TAGS").
		StringMap()
	proxyPrometheusPrefix = proxyCommand.Flag("prometheus-prefix",
		"Which namespace to use to send stats to Prometheus.").
		Envar("MTG_PROMETHEUS_PREFIX").
		Default("mtg").
		String()
	proxyWriteBufferSize = proxyCommand.Flag("write-buffer",
		"Write buffer size in bytes. You can think about it as a buffer from client to Telegram.").
		Short('w').
		Envar("MTG_BUFFER_WRITE").
		Default("65536").
		Uint32()
	proxyReadBufferSize = proxyCommand.Flag("read-buffer",
		"Read buffer size in bytes. You can think about it as a buffer from Telegram to client.").
		Short('r').
		Envar("MTG_BUFFER_READ").
		Default("131072").
		Uint32()
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
			config.Opt{Option: config.OptionTypeBindIP, Value: *proxyBindIP},
			config.Opt{Option: config.OptionTypeBindPort, Value: *proxyBindPort},
			config.Opt{Option: config.OptionTypePublicIPv4, Value: *proxyPublicIPv4},
			config.Opt{Option: config.OptionTypePublicIPv4Port, Value: *proxyPublicIPv4Port},
			config.Opt{Option: config.OptionTypePublicIPv6, Value: *proxyPublicIPv6},
			config.Opt{Option: config.OptionTypePublicIPv6Port, Value: *proxyPublicIPv6Port},
			config.Opt{Option: config.OptionTypeStatsIP, Value: *proxyStatsIP},
			config.Opt{Option: config.OptionTypeStatsPort, Value: *proxyStatsPort},
			config.Opt{Option: config.OptionTypeStatsdIP, Value: *proxyStatsdIP},
			config.Opt{Option: config.OptionTypeStatsdPort, Value: *proxyStatsdPort},
			config.Opt{Option: config.OptionTypeStatsdNetwork, Value: *proxyStatsdNetwork},
			config.Opt{Option: config.OptionTypeStatsdPrefix, Value: *proxyStatsdPrefix},
			config.Opt{Option: config.OptionTypeStatsdTagsFormat, Value: *proxyStatsdTagsFormat},
			config.Opt{Option: config.OptionTypeStatsdTags, Value: *proxyStatsdTags},
			config.Opt{Option: config.OptionTypePrometheusPrefix, Value: *proxyPrometheusPrefix},
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
