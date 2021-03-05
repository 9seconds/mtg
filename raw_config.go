package main

import (
	"fmt"
	"io"

	"github.com/pelletier/go-toml"
)

type rawConfig struct {
	Debug      bool   `toml:"debug"`
	Secret     string `toml:"secret"`
	BindTo     string `toml:"bind-to"`
	TCPBuffer  string `toml:"tcp-buffer"`
	PreferIP   string `toml:"prefer-ip"`
	CloakPort  uint   `toml:"cloak-port"`
	AccessFile string `toml:"access-file"`
	Probes     struct {
		Time struct {
			Enabled       bool   `toml:"enabled"`
			AllowSkewness string `toml:"allow-skewness"`
		} `toml:"time"`
		AntiReplay struct {
			Enabled bool   `toml:"enabled"`
			MaxSize string `toml:"max-size"`
			TTL     string `toml:"ttl"`
		} `toml:"anti-replay"`
	} `toml:"probes"`
	PublicIP struct {
		IPv4 string `toml:"ipv4"`
		IPv6 string `toml:"ipv6"`
	} `toml:"public-ip"`
	Dialers struct {
		Telegram string `toml:"telegram"`
		Default  string `toml:"default"`
	} `toml:"dialers"`
	Stats struct {
		StatsD struct {
			Enabled      bool   `toml:"enabled"`
			Address      string `toml:"address"`
			MetricPrefix string `toml:"metric-prefix"`
		} `toml:"statsd"`
		Prometheus struct {
			Enabled      bool   `toml:"enabled"`
			BindTo       string `toml:"bind-to"`
			HttpPath     string `toml:"http-path"`
			MetricPrefix string `toml:"metric-prefix"`
		} `toml:"prometheus"`
	} `toml:"stats"`
}

func parseRawConfig(reader io.Reader) (*rawConfig, error) {
	conf := &rawConfig{}

	if err := toml.NewDecoder(reader).Decode(conf); err != nil {
		return nil, fmt.Errorf("cannot parse config: %w", err)
	}

	return conf, nil
}
