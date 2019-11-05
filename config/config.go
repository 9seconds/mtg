package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/alecthomas/units"
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

	OptionTypeBind
	OptionTypePublicIPv4
	OptionTypePublicIPv6

	OptionTypeStatsBind
	OptionTypeStatsNamespace
	OptionTypeStatsdAddress
	OptionTypeStatsdNetwork
	OptionTypeStatsdTagsFormat
	OptionTypeStatsdTags

	OptionTypeWriteBufferSize
	OptionTypeReadBufferSize

	OptionTypeCloakPort

	OptionTypeAntiReplayMaxSize
	OptionTypeAntiReplayEvictionTime

	OptionTypeSecret
	OptionTypeAdtag
)

type Config struct {
	Bind       *net.TCPAddr `json:"bind"`
	PublicIPv4 *net.TCPAddr `json:"public_ipv4"`
	PublicIPv6 *net.TCPAddr `json:"public_ipv6"`
	StatsBind  *net.TCPAddr `json:"stats_bind"`
	StatsdAddr *net.TCPAddr `json:"stats_addr"`

	StatsNamespace string            `json:"stats_namespace"`
	StatsdNetwork  string            `json:"statsd_network"`
	CloakHost      string            `json:"cloak_host"`
	StatsdTags     map[string]string `json:"statsd_tags"`

	WriteBuffer int `json:"write_buffer"`
	ReadBuffer  int `json:"read_buffer"`
	CloakPort   int `json:"cloak_port"`

	AntiReplayMaxSize      int           `json:"anti_replay_max_size"`
	AntiReplayEvictionTime time.Duration `json:"anti_replay_eviction_time"`

	Debug            bool             `json:"debug"`
	Verbose          bool             `json:"verbose"`
	StatsdTagsFormat statsd.TagFormat `json:"statsd_tags_format"`
	SecretMode       SecretMode       `json:"secret_mode"`

	Secret []byte `json:"secret"`
	AdTag  []byte `json:"adtag"`
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
		case OptionTypeStatsdNetwork:
			value := opt.Value.(string)
			switch value {
			case "udp", "tcp":
				C.StatsdNetwork = value
			default:
				return fmt.Errorf("unknown statsd network %v", value)
			}
		case OptionTypeStatsdTagsFormat:
			value := opt.Value.(string)
			switch value {
			case "datadog":
				C.StatsdTagsFormat = statsd.Datadog
			case "influxdb":
				C.StatsdTagsFormat = statsd.InfluxDB
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
			C.AntiReplayMaxSize = opt.Value.(int)
		case OptionTypeAntiReplayEvictionTime:
			C.AntiReplayEvictionTime = opt.Value.(time.Duration)
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
	case len(C.Secret) == SimpleSecretLength:
		C.SecretMode = SecretModeSimple
	case bytes.HasPrefix(C.Secret, []byte{0xee}):
		C.SecretMode = SecretModeTLS
		secret := bytes.TrimPrefix(C.Secret, []byte{0xee})
		C.Secret = secret[:SimpleSecretLength]
		C.CloakHost = string(secret[SimpleSecretLength:])
	default:
		return errors.New("incorrect secret")
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
