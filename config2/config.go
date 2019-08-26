package config2

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/juju/errors"
	statsd "gopkg.in/alexcesaro/statsd.v2"
)

type SecretType byte

func (s SecretType) String() string {
	switch s {
	case SecretTypeMain:
		return "main"
	case SecretTypeSecured:
		return "secured"
	default:
		return "tls"
	}
}

const (
	SecretTypeMain = 1 << iota
	SecretTypeSecured
	SecretTypeTLS
)

const (
	FlagDebug   = "debug"
	FlagVerbose = "verbose"

	FlagBindIP         = "bind-ip"
	FlagBindPort       = "bind-port"
	FlagPublicIPv4     = "public-ipv4"
	FlagPublicIPv4Port = "public-ipv4-port"
	FlagPublicIPv6     = "public-ipv6"
	FlagPublicIPv6Port = "public-ipv6-port"
	FlagStatsIP        = "stats-ip"
	FlagStatsPort      = "stats-port"

	FlagStatsdIP         = "statsd-ip"
	FlagStatsdPort       = "statsd-port"
	FlagStatsdNetwork    = "statsd-network"
	FlagStatsdPrefix     = "statsd-prefix"
	FlagStatsdTagsFormat = "statsd-tags-format"
	FlagStatsdTags       = "statsd-tags"

	FlagPrometheusPrefix = "prometheus-prefix"

	FlagWriteBufferSize = "write-buffer"
	FlagReadBufferSize  = "read-buffer"

	FlagSecureOnly = "secure-only"

	FlagAntiReplayMaxSize      = "anti-replay-max-size"
	FlagAntiReplayEvictionTime = "anti-replay-eviction-time"

	FlagSecret = "secret"
	FlagAdtag  = "adtag"
)

type BufferSize struct {
	Read  int `json:"read"`
	Write int `json:"write"`
}

type AntiReplay struct {
	MaxSize      int           `json:"max_size"`
	EvictionTime time.Duration `json:"duration"`
}

type Stats struct {
	Prefix  string `json:"prefix"`
	Enabled bool   `json:"enabled"`
}

type StatsdStats struct {
	Stats

	Addr       Addr              `json:"addr"`
	Tags       map[string]string `json:"tags"`
	TagsFormat statsd.TagFormat  `json:"format"`
}

type PrometheusStats struct {
	Stats
}

type Addr struct {
	IP   net.IP `json:"ip"`
	Port int    `json:"port"`
	net  string
}

func (a Addr) Network() string {
	if a.net == "" {
		return "tcp"
	}
	return a.net
}

func (a Addr) String() string {
	return net.JoinHostPort(a.IP.String(), strconv.Itoa(a.Port))
}

func (a Addr) MarshalJSON() ([]byte, error) {
	data := map[string]string{
		"network": a.Network(),
		"addr":    a.String(),
	}
	return json.Marshal(data)
}

type Config struct {
	BufferSize BufferSize `json:"buffer_size"`
	AntiReplay AntiReplay `json:"anti_replay"`

	ListenAddr     Addr `json:"listen_addr"`
	PublicIPv4Addr Addr `json:"public_ipv4_addr"`
	PublicIPv6Addr Addr `json:"public_ipv6_addr"`
	StatsAddr      Addr `json:"stats_addr"`

	StatsdStats     StatsdStats     `json:"stats_statsd"`
	PrometheusStats PrometheusStats `json:"stats_prometheus"`

	Debug      bool       `json:"debug"`
	Verbose    bool       `json:"verbose"`
	SecureOnly bool       `json:"secure_only"`
	SecretType SecretType `json:"secret_type"`
	Secret     []byte     `json:"secret"`
	AdTag      []byte     `json:"adtag"`
}

func (c Config) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

type ConfigOpt struct {
	Name  string
	Value interface{}
}

var C = Config{}

func Init(options ...ConfigOpt) error { // nolint: gocyclo
	for _, opt := range options {
		switch opt.Name {
		case FlagDebug:
			C.Debug = opt.Value.(bool)
		case FlagVerbose:
			C.Verbose = opt.Value.(bool)
		case FlagBindIP:
			C.ListenAddr.IP = opt.Value.(net.IP)
		case FlagBindPort:
			C.ListenAddr.Port = opt.Value.(int)
		case FlagPublicIPv4:
			C.PublicIPv4Addr.IP = opt.Value.(net.IP)
		case FlagPublicIPv4Port:
			C.PublicIPv4Addr.Port = opt.Value.(int)
		case FlagPublicIPv6:
			C.PublicIPv6Addr.IP = opt.Value.(net.IP)
		case FlagPublicIPv6Port:
			C.PublicIPv6Addr.Port = opt.Value.(int)
		case FlagStatsIP:
			C.StatsAddr.IP = opt.Value.(net.IP)
		case FlagStatsPort:
			C.StatsAddr.Port = opt.Value.(int)
		case FlagStatsdIP:
			C.StatsdStats.Addr.IP = opt.Value.(net.IP)
		case FlagStatsdPort:
			C.StatsdStats.Addr.Port = opt.Value.(int)
		case FlagStatsdNetwork:
			C.StatsdStats.Addr.net = opt.Value.(string)
		case FlagStatsdPrefix:
			C.StatsdStats.Prefix = opt.Value.(string)
		case FlagStatsdTagsFormat:
			value := opt.Value.(string)
			switch value {
			case "datadog":
				C.StatsdStats.TagsFormat = statsd.Datadog
			case "influxdb":
				C.StatsdStats.TagsFormat = statsd.InfluxDB
			default:
				return errors.Errorf("Incorrect statsd tag %s", value)
			}
		case FlagStatsdTags:
			C.StatsdStats.Tags = opt.Value.(map[string]string)
		case FlagPrometheusPrefix:
			C.PrometheusStats.Prefix = opt.Value.(string)
		case FlagWriteBufferSize:
			C.BufferSize.Write = opt.Value.(int)
		case FlagReadBufferSize:
			C.BufferSize.Read = opt.Value.(int)
		case FlagAntiReplayMaxSize:
			C.AntiReplay.MaxSize = opt.Value.(int)
		case FlagAntiReplayEvictionTime:
			C.AntiReplay.EvictionTime = opt.Value.(time.Duration)
		case FlagSecureOnly:
			C.SecureOnly = opt.Value.(bool)
		case FlagSecret:
			C.Secret = opt.Value.([]byte)
		case FlagAdtag:
			C.AdTag = opt.Value.([]byte)
		}
	}

	var defaultStatsdTags statsd.TagFormat
	if C.StatsdStats.TagsFormat == defaultStatsdTags {
		C.StatsdStats.TagsFormat = statsd.Datadog
	}
	if C.StatsdStats.Addr.net == "" {
		C.StatsdStats.Addr.net = "udp"
	}

	switch {
	case len(C.Secret) == 17 && bytes.HasPrefix(C.Secret, []byte{0xdd}):
		C.SecretType = SecretTypeSecured
		C.Secret = bytes.TrimPrefix(C.Secret, []byte{0xdd})
	case len(C.Secret) == 16:
		C.SecretType = SecretTypeMain
	default:
		return errors.New("Incorrect secret")
	}

	return nil
}

func InitPublicAddress() error {
	if C.PublicIPv4Addr.Port == 0 {
		C.PublicIPv4Addr.Port = C.ListenAddr.Port
	}
	if C.PublicIPv6Addr.Port == 0 {
		C.PublicIPv6Addr.Port = C.ListenAddr.Port
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}
	done := make(chan struct{})

	if C.PublicIPv4Addr.IP == nil {
		wg.Add(1)
		go func() {
			getGlobalIPv4(ctx, cancel)
			wg.Done()
		}()
	}
	if C.PublicIPv6Addr.IP == nil {
		wg.Add(1)
		go func() {
			getGlobalIPv6(ctx, cancel)
			wg.Done()

		}()
	}
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
