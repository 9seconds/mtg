package config

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/pelletier/go-toml"
)

type tomlConfig struct {
	Debug                    bool   `toml:"debug" json:"debug,omitempty"`
	AllowFallbackOnUnknownDC bool   `toml:"allow-fallback-on-unknown-dc" json:"allowFallbackOnUnknownDc,omitempty"`
	Secret                   string `toml:"secret" json:"secret"`
	BindTo                   string `toml:"bind-to" json:"bindTo"`
	PreferIP                 string `toml:"prefer-ip" json:"preferIp,omitempty"`
	DomainFrontingPort       uint   `toml:"domain-fronting-port" json:"domainFrontingPort,omitempty"`
	TolerateTimeSkewness     string `toml:"tolerate-time-skewness" json:"tolerateTimeSkewness,omitempty"`
	Concurrency              uint   `toml:"concurrency" json:"concurrency,omitempty"`
	Defense                  struct {
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
		Allowlist struct {
			Enabled             bool     `toml:"enabled" json:"enabled,omitempty"`
			DownloadConcurrency uint     `toml:"download-concurrency" json:"downloadConcurrency,omitempty"`
			URLs                []string `toml:"urls" json:"urls,omitempty"`
			UpdateEach          string   `toml:"update-each" json:"updateEach,omitempty"`
		} `toml:"allowlist" json:"allowlist,omitempty"`
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
	tomlConf := &tomlConfig{}
	jsonBuf := &bytes.Buffer{}
	conf := &Config{}

	jsonEncoder := json.NewEncoder(jsonBuf)
	jsonEncoder.SetEscapeHTML(false)
	jsonEncoder.SetIndent("", "")

	if err := toml.Unmarshal(rawData, tomlConf); err != nil {
		return nil, fmt.Errorf("cannot parse toml config: %w", err)
	}

	if err := jsonEncoder.Encode(tomlConf); err != nil {
		panic(err)
	}

	if err := json.NewDecoder(jsonBuf).Decode(conf); err != nil {
		return nil, fmt.Errorf("cannot parse a config: %w", err)
	}

	return conf, nil
}
