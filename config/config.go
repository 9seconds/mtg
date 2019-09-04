package config

import (
	"bytes"
	"encoding/json"
	"net"
	"strconv"
	"time"

	"github.com/juju/errors"
	"go.uber.org/zap"
	statsd "gopkg.in/alexcesaro/statsd.v2"
)

type SecretMode uint8

func (s SecretMode) String() string {
	switch s {
	case SecretModeSimple:
		return "simple"
	case SecretModeSecured:
		return "secured"
	}
	return "tls"
}

const (
	SecretModeSimple SecretMode = iota
	SecretModeSecured
	SecretModeTLS
)

const SimpleSecretLength = 16

type OptionType uint8

const (
	OptionTypeDebug OptionType = iota
	OptionTypeVerbose

	OptionTypeBindIP
	OptionTypeBindPort
	OptionTypePublicIPv4
	OptionTypePublicIPv4Port
	OptionTypePublicIPv6
	OptionTypePublicIPv6Port
	OptionTypeStatsIP
	OptionTypeStatsPort

	OptionTypeStatsdIP
	OptionTypeStatsdPort
	OptionTypeStatsdNetwork
	OptionTypeStatsdPrefix
	OptionTypeStatsdTagsFormat
	OptionTypeStatsdTags
	OptionTypePrometheusPrefix

	OptionTypeWriteBufferSize
	OptionTypeReadBufferSize

	OptionTypeAntiReplayMaxSize
	OptionTypeAntiReplayEvictionTime

	OptionTypeSecret
	OptionTypeAdtag
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
	Prefix string `json:"prefix"`
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
	SecretMode SecretMode `json:"secret_mode"`
	Secret     []byte     `json:"secret"`
	AdTag      []byte     `json:"adtag"`
}

func (c Config) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

type ConfigOpt struct {
	Option OptionType
	Value  interface{}
}

var C = Config{}

func Init(options ...ConfigOpt) error { // nolint: gocyclo
	for _, opt := range options {
		switch opt.Option {
		case OptionTypeDebug:
			C.Debug = opt.Value.(bool)
		case OptionTypeVerbose:
			C.Verbose = opt.Value.(bool)
		case OptionTypeBindIP:
			C.ListenAddr.IP = opt.Value.(net.IP)
		case OptionTypeBindPort:
			C.ListenAddr.Port = int(opt.Value.(uint16))
		case OptionTypePublicIPv4:
			C.PublicIPv4Addr.IP = opt.Value.(net.IP)
		case OptionTypePublicIPv4Port:
			C.PublicIPv4Addr.Port = int(opt.Value.(uint16))
		case OptionTypePublicIPv6:
			C.PublicIPv6Addr.IP = opt.Value.(net.IP)
		case OptionTypePublicIPv6Port:
			C.PublicIPv6Addr.Port = int(opt.Value.(uint16))
		case OptionTypeStatsIP:
			C.StatsAddr.IP = opt.Value.(net.IP)
		case OptionTypeStatsPort:
			C.StatsAddr.Port = int(opt.Value.(uint16))
		case OptionTypeStatsdIP:
			C.StatsdStats.Addr.IP = opt.Value.(net.IP)
		case OptionTypeStatsdPort:
			C.StatsdStats.Addr.Port = int(opt.Value.(uint16))
		case OptionTypeStatsdNetwork:
			C.StatsdStats.Addr.net = opt.Value.(string)
		case OptionTypeStatsdPrefix:
			C.StatsdStats.Prefix = opt.Value.(string)
		case OptionTypeStatsdTagsFormat:
			value := opt.Value.(string)
			switch value {
			case "datadog":
				C.StatsdStats.TagsFormat = statsd.Datadog
			case "influxdb":
				C.StatsdStats.TagsFormat = statsd.InfluxDB
			default:
				return errors.Errorf("Incorrect statsd tag %s", value)
			}
		case OptionTypeStatsdTags:
			C.StatsdStats.Tags = opt.Value.(map[string]string)
		case OptionTypePrometheusPrefix:
			C.PrometheusStats.Prefix = opt.Value.(string)
		case OptionTypeWriteBufferSize:
			C.BufferSize.Write = int(opt.Value.(uint32))
		case OptionTypeReadBufferSize:
			C.BufferSize.Read = int(opt.Value.(uint32))
		case OptionTypeAntiReplayMaxSize:
			C.AntiReplay.MaxSize = opt.Value.(int)
		case OptionTypeAntiReplayEvictionTime:
			C.AntiReplay.EvictionTime = opt.Value.(time.Duration)
		case OptionTypeSecret:
			C.Secret = opt.Value.([]byte)
		case OptionTypeAdtag:
			C.AdTag = opt.Value.([]byte)
		default:
			return errors.Errorf("Unknown tag %v", opt.Option)
		}
	}

	switch {
	case len(C.Secret) == 1+SimpleSecretLength && bytes.HasPrefix(C.Secret, []byte{0xdd}):
		C.SecretMode = SecretModeSecured
		C.Secret = bytes.TrimPrefix(C.Secret, []byte{0xdd})
	case len(C.Secret) == SimpleSecretLength:
		C.SecretMode = SecretModeSimple
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

	foundAddress := C.PublicIPv4Addr.IP != nil || C.PublicIPv6Addr.IP != nil
	if C.PublicIPv4Addr.IP == nil {
		ip, err := getGlobalIPv4()
		if err != nil {
			zap.S().Warnw("Cannot resolve public address", "error", err)
		} else {
			C.PublicIPv4Addr.IP = ip
			foundAddress = true
		}
	}
	if C.PublicIPv6Addr.IP == nil {
		ip, err := getGlobalIPv6()
		if err != nil {
			zap.S().Warnw("Cannot resolve public address", "error", err)
		} else {
			C.PublicIPv6Addr.IP = ip
			foundAddress = true
		}
	}

	if !foundAddress {
		return errors.New("Cannot resolve any public address")
	}

	return nil
}
