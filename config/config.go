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
	Probes    struct {
		Time struct {
			Enabled       bool         `json:"enabled"`
			AllowSkewness TypeDuration `json:"allow-skewness"`
		} `json:"time"`
		AntiReplay struct {
			Enabled   bool          `json:"enabled"`
			MaxSize   TypeBytes     `json:"max-size"`
			ErrorRate TypeErrorRate `json:"error-rate"`
		} `json:"anti-replay"`
	} `json:"probes"`
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
	if len(c.Secret.Key) == 0 || c.Secret.Host == "" {
		return fmt.Errorf("incorrect secret %s", c.Secret.String())
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
	Debug     bool   `toml:"debug" json:"debug"`
	Secret    string `toml:"secret" json:"secret"`
	BindTo    string `toml:"bind-to" json:"bind-to"`
	TCPBuffer string `toml:"tcp-buffer" json:"tcp-buffer"`
	PreferIP  string `toml:"prefer-ip" json:"prefer-ip"`
	CloakPort uint   `toml:"cloak-port" json:"cloak-port"`
	Probes    struct {
		Time struct {
			Enabled       bool   `toml:"enabled" json:"enabled"`
			AllowSkewness string `toml:"allow-skewness" json:"allow-skewness"`
		} `toml:"time" json:"time"`
		AntiReplay struct {
			Enabled   bool    `toml:"enabled" json:"enabled"`
			MaxSize   string  `toml:"max-size" json:"max-size"`
			ErrorRate float64 `toml:"error-rate" json:"error-rate"`
		} `toml:"anti-replay" json:"anti-replay"`
	} `toml:"probes" json:"probes"`
	Network struct {
		PublicIP struct {
			IPv4 string `toml:"ipv4" json:"ipv4"`
			IPv6 string `toml:"ipv6" json:"ipv6"`
		} `toml:"public-ip" json:"public-ip"`
		Timeout struct {
			TCP  string `toml:"tcp" json:"tcp"`
			Idle string `toml:"idle" json:"idle"`
		} `toml:"timeout" json:"timeout"`
		DOHIP   string   `toml:"doh-ip" json:"doh-ip"`
		Proxies []string `toml:"proxies" json:"proxies"`
	} `toml:"network" json:"network"`
	Stats struct {
		StatsD struct {
			Enabled      bool   `toml:"enabled" json:"enabled"`
			Address      string `toml:"address" json:"address"`
			MetricPrefix string `toml:"metric-prefix" json:"metric-prefix"`
		} `toml:"statsd" json:"statsd"`
		Prometheus struct {
			Enabled      bool   `toml:"enabled" json:"enabled"`
			BindTo       string `toml:"bind-to" json:"bind-to"`
			HTTPPath     string `toml:"http-path" json:"http-path"`
			MetricPrefix string `toml:"metric-prefix" json:"metric-prefix"`
		} `toml:"prometheus" json:"prometheus"`
	} `toml:"stats" json:"stats"`
}

func Parse(rawData []byte) (*Config, error) {
	rawConf := &configRaw{}

	if err := toml.Unmarshal(rawData, rawConf); err != nil {
		return nil, fmt.Errorf("cannot parse toml config: %w", err)
	}

	jsonBuf := &bytes.Buffer{}
	jsonEncoder := json.NewEncoder(jsonBuf)

	jsonEncoder.SetEscapeHTML(false)
	jsonEncoder.SetIndent("", "")

	if err := jsonEncoder.Encode(rawConf); err != nil {
		return nil, fmt.Errorf("cannot dump into interim format: %w", err)
	}

	conf := &Config{}

	if err := json.NewDecoder(jsonBuf).Decode(conf); err != nil {
		return nil, fmt.Errorf("cannot parse final config: %w", err)
	}

	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("cannot validate config: %w", err)
	}

	return conf, nil
}
