package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/pelletier/go-toml"
)

type config struct {
	Debug  bool          `json:"debug"`
	Secret mtglib.Secret `json:"secret"`
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
			Enabled bool   `toml:"enabled" json:"enabled"`
			MaxSize string `toml:"max-size" json:"max-size"`
			TTL     string `toml:"ttl" json:"ttl"`
		} `toml:"anti-replay" json:"anti-replay"`
	} `toml:"probes" json:"probes"`
	Network struct {
		PublicIP struct {
			IPv4 string `toml:"ipv4" json:"ipv4"`
			IPv6 string `toml:"ipv6" json:"ipv6"`
		} `toml:"public-ip" json:"public-ip"`
		DOHHostname string   `toml:"doh-hostname" json:"doh-hostname"`
		Proxies     []string `toml:"proxies" json:"proxies"`
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

func parseConfig(reader io.Reader) (*config, error) {
	rawConf := &configRaw{}

	if err := toml.NewDecoder(reader).Decode(rawConf); err != nil {
		return nil, fmt.Errorf("cannot parse toml config: %w", err)
	}

	jsonBuf := &bytes.Buffer{}
	jsonEncoder := json.NewEncoder(jsonBuf)

	jsonEncoder.SetEscapeHTML(false)
	jsonEncoder.SetIndent("", "")

	if err := jsonEncoder.Encode(rawConf); err != nil {
		return nil, fmt.Errorf("cannot dump into interim format: %w", err)
	}

	conf := &config{}

	if err := json.NewDecoder(jsonBuf).Decode(conf); err != nil {
		return nil, fmt.Errorf("cannot parse final config: %w", err)
	}

	return conf, nil
}
