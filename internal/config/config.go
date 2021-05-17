package config

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/pelletier/go-toml"
)

type Config struct {
	Debug                bool          `json:"debug"`
	Secret               mtglib.Secret `json:"secret"`
	BindTo               TypeHostPort  `json:"bindTo"`
	TCPBuffer            TypeBytes     `json:"tcpBuffer"`
	PreferIP             TypePreferIP  `json:"preferIp"`
	DomainFrontingPort   TypePort      `json:"domainFrontingPort"`
	TolerateTimeSkewness TypeDuration  `json:"tolerateTimeSkewness"`
	Concurrency          uint          `json:"concurrency"`
	Defense              struct {
		AntiReplay struct {
			Enabled   bool          `json:"enabled"`
			MaxSize   TypeBytes     `json:"maxSize"`
			ErrorRate TypeErrorRate `json:"errorRate"`
		} `json:"antiReplay"`
		Blocklist struct {
			Enabled             bool               `json:"enabled"`
			DownloadConcurrency uint               `json:"downloadConcurrency"`
			URLs                []TypeBlocklistURI `json:"urls"`
			UpdateEach          TypeDuration       `json:"updateEach"`
		} `json:"blocklist"`
	} `json:"defense"`
	Network struct {
		Timeout struct {
			TCP  TypeDuration `json:"tcp"`
			HTTP TypeDuration `json:"http"`
			Idle TypeDuration `json:"idle"`
		} `json:"timeout"`
		DOHIP   TypeIP    `json:"dohIp"`
		Proxies []TypeURL `json:"proxies"`
	} `json:"network"`
	Stats struct {
		StatsD struct {
			Enabled      bool                `json:"enabled"`
			Address      TypeHostPort        `json:"address"`
			MetricPrefix TypeMetricPrefix    `json:"metricPrefix"`
			TagFormat    TypeStatsdTagFormat `json:"tagFormat"`
		} `json:"statsd"`
		Prometheus struct {
			Enabled      bool             `json:"enabled"`
			BindTo       TypeHostPort     `json:"bindTo"`
			HTTPPath     TypeHTTPPath     `json:"httpPath"`
			MetricPrefix TypeMetricPrefix `json:"metricPrefix"`
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
	Debug                bool   `toml:"debug" json:"debug,omitempty"`
	Secret               string `toml:"secret" json:"secret"`
	BindTo               string `toml:"bind-to" json:"bindTo"`
	TCPBuffer            string `toml:"tcp-buffer" json:"tcpBuffer,omitempty"`
	PreferIP             string `toml:"prefer-ip" json:"preferIp,omitempty"`
	DomainFrontingPort   uint   `toml:"domain-fronting-port" json:"domainFrontingPort,omitempty"`
	TolerateTimeSkewness string `toml:"tolerate-time-skewness" json:"tolerateTimeSkewness,omitempty"`
	Concurrency          uint   `toml:"concurrency" json:"concurrency,omitempty"`
	Defense              struct {
		AntiReplay struct {
			Enabled   bool    `toml:"enabled" json:"enabled,omitempty"`
			MaxSize   string  `toml:"max-size" json:"maxSize,omitempty"`
			ErrorRate float64 `toml:"error-rate" json:"errorRate,omitempty"`
		} `toml:"anti-replay" json:"antiReplay,omitempty"`
		Blocklist struct {
			Enabled             bool     `toml:"enabled" json:"enabled,omitempty"`
			DownloadConcurrency uint     `toml:"download-concurrency" json:"downloadConcurrency,omitempty"`
			URLs                []string `toml:"urls" json:"urls,omitempty"`
			UpdateEach          string   `toml:"update-each" json:"updateEach,omitempty"`
		} `toml:"blocklist" json:"blocklist,omitempty"`
	} `toml:"defense" json:"defense,omitempty"`
	Network struct {
		Timeout struct {
			TCP  string `toml:"tcp" json:"tcp,omitempty"`
			HTTP string `toml:"http" json:"http,omitempty"`
			Idle string `toml:"idle" json:"idle,omitempty"`
		} `toml:"timeout" json:"timeout,omitempty"`
		DOHIP   string   `toml:"doh-ip" json:"dohIp,omitempty"`
		Proxies []string `toml:"proxies" json:"proxies,omitempty"`
	} `toml:"network" json:"network,omitempty"`
	Stats struct {
		StatsD struct {
			Enabled      bool   `toml:"enabled" json:"enabled,omitempty"`
			Address      string `toml:"address" json:"address,omitempty"`
			MetricPrefix string `toml:"metric-prefix" json:"metricPrefix,omitempty"`
			TagFormat    string `toml:"tag-format" json:"tagFormat,omitempty"`
		} `toml:"statsd" json:"statsd,omitempty"`
		Prometheus struct {
			Enabled      bool   `toml:"enabled" json:"enabled,omitempty"`
			BindTo       string `toml:"bind-to" json:"bindTo,omitempty"`
			HTTPPath     string `toml:"http-path" json:"httpPath,omitempty"`
			MetricPrefix string `toml:"metric-prefix" json:"metricPrefix,omitempty"`
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
		panic(err)
	}

	if err := json.NewDecoder(jsonBuf).Decode(conf); err != nil {
		return nil, fmt.Errorf("cannot parse a config: %w", err)
	}

	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("cannot validate config: %w", err)
	}

	return conf, nil
}
