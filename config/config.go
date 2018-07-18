package config

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/juju/errors"
	statsd "gopkg.in/alexcesaro/statsd.v2"
)

// Buffer sizes define internal socket buffer sizes.
const (
	BufferWriteSize = 32 * 1024
	BufferReadSize  = 32 * 1024
)

// Config represents common configuration of mtg.
type Config struct {
	Debug      bool
	Verbose    bool
	SecureMode bool

	BindPort       uint16
	PublicIPv4Port uint16
	PublicIPv6Port uint16
	StatsPort      uint16

	BindIP     net.IP
	PublicIPv4 net.IP
	PublicIPv6 net.IP
	StatsIP    net.IP

	StatsD struct {
		Addr       net.Addr
		Prefix     string
		Tags       map[string]string
		TagsFormat statsd.TagFormat
		Enabled    bool
	}

	Secret []byte
	AdTag  []byte
}

// URLs contains links to the proxy (tg://, t.me) and their QR codes.
type URLs struct {
	TG        string `json:"tg_url"`
	TMe       string `json:"tme_url"`
	TGQRCode  string `json:"tg_qrcode"`
	TMeQRCode string `json:"tme_qrcode"`
}

// IPURLs contains links to both ipv4 and ipv6 of the proxy.
type IPURLs struct {
	IPv4      URLs   `json:"ipv4"`
	IPv6      URLs   `json:"ipv6"`
	BotSecret string `json:"secret_for_mtproxybot"`
}

// BindAddr returns connection for this server to bind to.
func (c *Config) BindAddr() string {
	return getAddr(c.BindIP, c.BindPort)
}

// StatAddr returns connection string to the stats API.
func (c *Config) StatAddr() string {
	return getAddr(c.StatsIP, c.StatsPort)
}

// UseMiddleProxy defines if this proxy has to connect middle proxies
// which supports promoted channels or directly access Telegram.
func (c *Config) UseMiddleProxy() bool {
	return len(c.AdTag) > 0
}

// BotSecretString returns secret string which should work with MTProxybot.
func (c *Config) BotSecretString() string {
	return hex.EncodeToString(c.Secret)
}

// SecretString returns a secret in a form entered on the start of the
// application.
func (c *Config) SecretString() string {
	secret := c.BotSecretString()
	if c.SecureMode {
		return "dd" + secret
	}
	return secret
}

// GetURLs returns configured IPURLs instance with links to this server.
func (c *Config) GetURLs() IPURLs {
	urls := IPURLs{}
	secret := c.SecretString()
	if c.PublicIPv4 != nil {
		urls.IPv4 = getURLs(c.PublicIPv4, c.PublicIPv4Port, secret)
	}
	if c.PublicIPv6 != nil {
		urls.IPv6 = getURLs(c.PublicIPv6, c.PublicIPv6Port, secret)
	}
	urls.BotSecret = c.BotSecretString()

	return urls
}

func getAddr(host fmt.Stringer, port uint16) string {
	return net.JoinHostPort(host.String(), strconv.Itoa(int(port)))
}

// NewConfig returns new configuration. If required, it manages and
// fetches data from external sources. Parameters passed to this
// function, should come from command line arguments.
func NewConfig(debug, verbose bool, // nolint: gocyclo
	bindIP, publicIPv4, publicIPv6, statsIP net.IP,
	bindPort, publicIPv4Port, publicIPv6Port, statsPort, statsdPort uint16,
	secret, adtag, statsdIP, statsdNetwork, statsdPrefix, statsdTagsFormat string,
	statsdTags map[string]string) (*Config, error) {
	secureMode := false
	if strings.HasPrefix(secret, "dd") && len(secret) == 34 {
		secureMode = true
		secret = strings.TrimPrefix(secret, "dd")
	} else if len(secret) != 32 {
		return nil, errors.New("Telegram demands secret of length 32")
	}
	secretBytes, err := hex.DecodeString(secret)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create config")
	}

	var adTagBytes []byte
	if len(adtag) != 0 {
		adTagBytes, err = hex.DecodeString(adtag)
		if err != nil {
			return nil, errors.Annotate(err, "Cannot create config")
		}
	}

	if publicIPv4 == nil {
		publicIPv4, err = getGlobalIPv4()
		if err != nil {
			publicIPv4 = nil
		} else if publicIPv4.To4() == nil {
			return nil, errors.Errorf("IP %s is not IPv4", publicIPv4.String())
		}
	}
	if publicIPv4Port == 0 {
		publicIPv4Port = bindPort
	}

	if publicIPv6 == nil {
		publicIPv6, err = getGlobalIPv6()
		if err != nil {
			publicIPv6 = nil
		} else if publicIPv6.To4() != nil {
			return nil, errors.Errorf("IP %s is not IPv6", publicIPv6.String())
		}
	}
	if publicIPv6Port == 0 {
		publicIPv6Port = bindPort
	}

	if statsIP == nil {
		statsIP = publicIPv4
	}

	conf := &Config{
		Debug:          debug,
		Verbose:        verbose,
		BindIP:         bindIP,
		BindPort:       bindPort,
		PublicIPv4:     publicIPv4,
		PublicIPv4Port: publicIPv4Port,
		PublicIPv6:     publicIPv6,
		PublicIPv6Port: publicIPv6Port,
		StatsIP:        statsIP,
		StatsPort:      statsPort,
		Secret:         secretBytes,
		AdTag:          adTagBytes,
		SecureMode:     secureMode,
	}

	if statsdIP != "" {
		conf.StatsD.Enabled = true
		conf.StatsD.Prefix = statsdPrefix
		conf.StatsD.Tags = statsdTags

		var addr net.Addr
		hostPort := net.JoinHostPort(statsdIP, strconv.Itoa(int(statsdPort)))
		switch statsdNetwork {
		case "tcp":
			addr, err = net.ResolveTCPAddr("tcp", hostPort)
		case "udp":
			addr, err = net.ResolveUDPAddr("udp", hostPort)
		default:
			err = errors.Errorf("Unknown network %s", statsdNetwork)
		}
		if err != nil {
			return nil, errors.Annotate(err, "Cannot resolve statsd address")
		}
		conf.StatsD.Addr = addr

		switch statsdTagsFormat {
		case "datadog":
			conf.StatsD.TagsFormat = statsd.Datadog
		case "influxdb":
			conf.StatsD.TagsFormat = statsd.InfluxDB
		case "":
		default:
			return nil, errors.Errorf("Unknown tags format %s", statsdTagsFormat)
		}
	}

	return conf, nil
}
