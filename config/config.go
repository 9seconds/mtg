package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"

	"github.com/alecthomas/units"
	statsd "github.com/smira/go-statsd"
	"go.uber.org/zap"
)

type SecretMode uint8

func (s SecretMode) String() string {
	switch s {
	case SecretModeSimple:
		return "simple"
	case SecretModeSecured:
		return "secured"
	case SecretModeTLS:
		return "tls"
	}

	return "tls"
}

const (
	SecretModeSimple SecretMode = iota
	SecretModeSecured
	SecretModeTLS
)

type PreferIP uint8

const (
	PreferIPv4 PreferIP = iota
	PreferIPv6
)

const SimpleSecretLength = 16

type OptionType uint8

const (
	OptionTypeDebug OptionType = iota
	OptionTypeVerbose

	OptionTypePreferIP

	OptionTypeBind
	OptionTypePublicIPv4
	OptionTypePublicIPv6

	OptionTypeStatsBind
	OptionTypeStatsNamespace
	OptionTypeStatsdAddress
	OptionTypeStatsdTagsFormat
	OptionTypeStatsdTags

	OptionTypeWriteBufferSize
	OptionTypeReadBufferSize

	OptionTypeCloakPort

	OptionTypeAntiReplayMaxSize

	OptionTypeMultiplexPerConnection

	OptionTypeNTPServers

	OptionTypeSecret
	OptionTypeAdtag
)

type Config struct {
	Bind             *net.TCPAddr      `json:"bind"`
	PublicIPv4       *net.TCPAddr      `json:"public_ipv4"`
	PublicIPv6       *net.TCPAddr      `json:"public_ipv6"`
	StatsBind        *net.TCPAddr      `json:"stats_bind"`
	StatsdAddr       *net.TCPAddr      `json:"stats_addr"`
	StatsdTagsFormat *statsd.TagFormat `json:"statsd_tags_format"`

	StatsNamespace string            `json:"stats_namespace"`
	CloakHost      string            `json:"cloak_host"`
	StatsdTags     map[string]string `json:"statsd_tags"`

	WriteBuffer int `json:"write_buffer"`
	ReadBuffer  int `json:"read_buffer"`
	CloakPort   int `json:"cloak_port"`

	AntiReplayMaxSize int `json:"anti_replay_max_size"`

	MultiplexPerConnection int `json:"multiplex_per_connection"`

	Debug      bool       `json:"debug"`
	Verbose    bool       `json:"verbose"`
	SecretMode SecretMode `json:"secret_mode"`
	PreferIP   PreferIP   `json:"prefer_ip"`
	NTPServers []string   `json:"ntp_servers"`

	Secret []byte `json:"secret"`
	AdTag  []byte `json:"adtag"`
}

func (c *Config) ClientReadBuffer() int {
	return c.ReadBuffer
}

func (c *Config) ClientWriteBuffer() int {
	return c.WriteBuffer
}

func (c *Config) MiddleProxyMode() bool {
	return len(c.AdTag) > 0
}

func (c *Config) ProxyReadBuffer() int {
	value := c.ReadBuffer

	if c.MiddleProxyMode() {
		value = c.adjustProxyValue(value)
	}

	return value
}

func (c *Config) ProxyWriteBuffer() int {
	value := c.WriteBuffer

	if c.MiddleProxyMode() {
		value = c.adjustProxyValue(value)
	}

	return value
}

func (c *Config) adjustProxyValue(value int) int {
	if c.MultiplexPerConnection == 0 {
		return value
	}

	fvalue := float64(value)

	newValue := fvalue * 2 * math.Log(float64(c.MultiplexPerConnection))
	newValue = math.Ceil(newValue)
	newValue = math.Max(fvalue, newValue)

	return int(newValue)
}

type Opt struct {
	Option OptionType
	Value  interface{}
}

var C = Config{}

func Init(options ...Opt) error { // nolint: gocyclo, funlen
	for _, opt := range options {
		switch opt.Option {
		case OptionTypeDebug:
			C.Debug = opt.Value.(bool)
		case OptionTypeVerbose:
			C.Verbose = opt.Value.(bool)
		case OptionTypePreferIP:
			value := opt.Value.(string)
			switch value {
			case "ipv4":
				C.PreferIP = PreferIPv4
			case "ipv6":
				C.PreferIP = PreferIPv6
			default:
				return fmt.Errorf("incorrect direct IP mode %s", value)
			}
		case OptionTypeBind:
			C.Bind = opt.Value.(*net.TCPAddr)
		case OptionTypePublicIPv4:
			C.PublicIPv4 = opt.Value.(*net.TCPAddr)
			if C.PublicIPv4 == nil {
				C.PublicIPv4 = &net.TCPAddr{}
			}
		case OptionTypePublicIPv6:
			C.PublicIPv6 = opt.Value.(*net.TCPAddr)
			if C.PublicIPv6 == nil {
				C.PublicIPv6 = &net.TCPAddr{}
			}
		case OptionTypeStatsBind:
			C.StatsBind = opt.Value.(*net.TCPAddr)
		case OptionTypeStatsNamespace:
			C.StatsNamespace = opt.Value.(string)
		case OptionTypeStatsdAddress:
			C.StatsdAddr = opt.Value.(*net.TCPAddr)
		case OptionTypeStatsdTagsFormat:
			value := opt.Value.(string)
			switch value {
			case "datadog":
				C.StatsdTagsFormat = statsd.TagFormatDatadog
			case "influxdb":
				C.StatsdTagsFormat = statsd.TagFormatInfluxDB
			default:
				return fmt.Errorf("incorrect statsd tag %s", value)
			}
		case OptionTypeStatsdTags:
			C.StatsdTags = opt.Value.(map[string]string)
		case OptionTypeWriteBufferSize:
			C.WriteBuffer = int(opt.Value.(units.Base2Bytes))
		case OptionTypeReadBufferSize:
			C.ReadBuffer = int(opt.Value.(units.Base2Bytes))
		case OptionTypeCloakPort:
			C.CloakPort = int(opt.Value.(uint16))
		case OptionTypeAntiReplayMaxSize:
			C.AntiReplayMaxSize = int(opt.Value.(units.Base2Bytes))
		case OptionTypeMultiplexPerConnection:
			C.MultiplexPerConnection = int(opt.Value.(uint))
		case OptionTypeNTPServers:
			C.NTPServers = opt.Value.([]string)
			if len(C.NTPServers) == 0 {
				return errors.New("ntp server list is empty")
			}
		case OptionTypeSecret:
			C.Secret = opt.Value.([]byte)
		case OptionTypeAdtag:
			C.AdTag = opt.Value.([]byte)
		default:
			return fmt.Errorf("unknown tag %v", opt.Option)
		}
	}

	switch {
	case len(C.Secret) == 1+SimpleSecretLength && bytes.HasPrefix(C.Secret, []byte{0xdd}):
		C.SecretMode = SecretModeSecured
		C.Secret = bytes.TrimPrefix(C.Secret, []byte{0xdd})
	case len(C.Secret) > SimpleSecretLength && bytes.HasPrefix(C.Secret, []byte{0xee}):
		C.SecretMode = SecretModeTLS
		secret := bytes.TrimPrefix(C.Secret, []byte{0xee})
		C.Secret = secret[:SimpleSecretLength]
		C.CloakHost = string(secret[SimpleSecretLength:])
	case len(C.Secret) == SimpleSecretLength:
		C.SecretMode = SecretModeSimple
	default:
		return errors.New("incorrect secret")
	}

	if C.MultiplexPerConnection == 0 {
		return errors.New("cannot use 0 clients per connection for multiplexing")
	}

	if C.CloakHost != "" {
		if _, err := net.LookupHost(C.CloakHost); err != nil {
			zap.S().Warnw("Cannot resolve address of host", "hostname", C.CloakHost, "error", err)
		}
	}

	return nil
}

func InitPublicAddress(ctx context.Context) error {
	if C.PublicIPv4.Port == 0 {
		C.PublicIPv4.Port = C.Bind.Port
	}

	if C.PublicIPv6.Port == 0 {
		C.PublicIPv6.Port = C.Bind.Port
	}

	foundAddress := C.PublicIPv4.IP != nil || C.PublicIPv6.IP != nil

	if C.PublicIPv4.IP == nil {
		ip, err := getGlobalIPv4(ctx)
		if err != nil {
			zap.S().Warnw("Cannot resolve public address", "error", err)
		} else {
			C.PublicIPv4.IP = ip
			foundAddress = true
		}
	}

	if C.PublicIPv6.IP == nil {
		ip, err := getGlobalIPv6(ctx)
		if err != nil {
			zap.S().Warnw("Cannot resolve public address", "error", err)
		} else {
			C.PublicIPv6.IP = ip
			foundAddress = true
		}
	}

	if !foundAddress {
		return errors.New("cannot resolve any public address")
	}

	return nil
}

func Printable() interface{} {
	data, err := json.Marshal(C)
	if err != nil {
		panic(err)
	}

	rv := map[string]interface{}{}
	if err := json.Unmarshal(data, &rv); err != nil {
		panic(err)
	}

	return rv
}
