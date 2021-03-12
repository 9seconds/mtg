package config

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/pelletier/go-toml"
)

type Config struct {
	Debug     bool          `json:"debug"`
	Secret    mtglib.Secret `json:"secret"`
	BindTo    TypeHostPort  `json:"bind-to"`
	TCPBuffer TypeBytes     `json:"tcp-buffer"`
	PreferIP  TypePreferIP  `json:"prefer-ip"`
	CloakPort TypePort      `json:"cloak-port"`
	Defense   struct {
		Time struct {
			Enabled       bool         `json:"enabled"`
			AllowSkewness TypeDuration `json:"allow-skewness"`
		} `json:"time"`
		AntiReplay struct {
			Enabled   bool          `json:"enabled"`
			MaxSize   TypeBytes     `json:"max-size"`
			ErrorRate TypeErrorRate `json:"error-rate"`
		} `json:"anti-replay"`
	} `json:"defense"`
	Network struct {
		PublicIP struct {
			IPv4 TypeIP `json:"ipv4"`
			IPv6 TypeIP `json:"ipv6"`
		} `json:"public-ip"`
		Timeout struct {
			TCP  TypeDuration `json:"tcp"`
			Idle TypeDuration `json:"idle"`
		} `json:"timeout"`
		DOHIP   TypeIP    `json:"doh-ip"`
		Proxies []TypeURL `json:"proxies"`
	} `json:"network"`
	Stats struct {
		StatsD struct {
			Enabled      bool             `json:"enabled"`
			Address      TypeHostPort     `json:"address"`
			MetricPrefix TypeMetricPrefix `json:"metric-prefix"`
		} `json:"statsd"`
		Prometheus struct {
			Enabled      bool             `json:"enabled"`
			BindTo       TypeHostPort     `json:"bind-to"`
			HTTPPath     TypeHTTPPath     `json:"http-path"`
			MetricPrefix TypeMetricPrefix `json:"metric-prefix"`
		} `json:"prometheus"`
	} `json:"stats"`
}

func (c *Config) Validate() error {
	if !c.Secret.Valid() {
		return fmt.Errorf("invalid secret %s", c.Secret.String())
	}
	if len(c.BindTo.HostValue(nil)) == 0 || c.BindTo.PortValue(0) == 0 {
		return fmt.Errorf("incorrect bind-to parameter %s", c.BindTo.String())
	}

	return nil
}

func (c *Config) String() string {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)

	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(c); err != nil {
		panic(err)
	}

	return buf.String()
}

type configRaw struct {
	Debug     bool   `toml:"debug" json:"debug,omitempty"`
	Secret    string `toml:"secret" json:"secret"`
	BindTo    string `toml:"bind-to" json:"bind-to"`
	TCPBuffer string `toml:"tcp-buffer" json:"tcp-buffer,omitempty"`
	PreferIP  string `toml:"prefer-ip" json:"prefer-ip,omitempty"`
	CloakPort uint   `toml:"cloak-port" json:"cloak-port,omitempty"`
	Defense   struct {
		Time struct {
			Enabled       bool   `toml:"enabled" json:"enabled,omitempty"`
			AllowSkewness string `toml:"allow-skewness" json:"allow-skewness,omitempty"`
		} `toml:"time" json:"time,omitempty"`
		AntiReplay struct {
			Enabled   bool    `toml:"enabled" json:"enabled,omitempty"`
			MaxSize   string  `toml:"max-size" json:"max-size,omitempty"`
			ErrorRate float64 `toml:"error-rate" json:"error-rate,omitempty"`
		} `toml:"anti-replay" json:"anti-replay,omitempty"`
	} `toml:"defense" json:"defense,omitempty"`
	Network struct {
		PublicIP struct {
			IPv4 string `toml:"ipv4" json:"ipv4,omitempty"`
			IPv6 string `toml:"ipv6" json:"ipv6,omitempty"`
		} `toml:"public-ip" json:"public-ip,omitempty"`
		Timeout struct {
			TCP  string `toml:"tcp" json:"tcp,omitempty"`
			Idle string `toml:"idle" json:"idle,omitempty"`
		} `toml:"timeout" json:"timeout,omitempty"`
		DOHIP   string   `toml:"doh-ip" json:"doh-ip,omitempty"`
		Proxies []string `toml:"proxies" json:"proxies,omitempty"`
	} `toml:"network" json:"network,omitempty"`
	Stats struct {
		StatsD struct {
			Enabled      bool   `toml:"enabled" json:"enabled,omitempty"`
			Address      string `toml:"address" json:"address,omitempty"`
			MetricPrefix string `toml:"metric-prefix" json:"metric-prefix,omitempty"`
		} `toml:"statsd" json:"statsd,omitempty"`
		Prometheus struct {
			Enabled      bool   `toml:"enabled" json:"enabled,omitempty"`
			BindTo       string `toml:"bind-to" json:"bind-to,omitempty"`
			HTTPPath     string `toml:"http-path" json:"http-path,omitempty"`
			MetricPrefix string `toml:"metric-prefix" json:"metric-prefix,omitempty"`
		} `toml:"prometheus" json:"prometheus,omitempty"`
	} `toml:"stats" json:"stats,omitempty"`
}

func Parse(rawData []byte) (*Config, error) {
	rawConf := &configRaw{}
	jsonBuf := &bytes.Buffer{}
	conf := &Config{}

	jsonEncoder := json.NewEncoder(jsonBuf)
	jsonEncoder.SetEscapeHTML(false)
	jsonEncoder.SetIndent("", "")

	if err := toml.Unmarshal(rawData, rawConf); err != nil {
		return nil, fmt.Errorf("cannot parse toml config: %w", err)
	}

	if err := jsonEncoder.Encode(rawConf); err != nil {
		return nil, fmt.Errorf("cannot dump into interim format: %w", err)
	}

	if err := json.NewDecoder(jsonBuf).Decode(conf); err != nil {
		return nil, fmt.Errorf("cannot parse final config: %w", err)
	}

	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("cannot validate config: %w", err)
	}

	return conf, nil
}
