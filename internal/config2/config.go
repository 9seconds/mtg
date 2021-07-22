package config2

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/9seconds/mtg/v2/mtglib"
)

type Config struct {
	Debug                TypeBool        `json:"debug"`
	Secret               mtglib.Secret   `json:"secret"`
	BindTo               TypeHostPort    `json:"bindTo"`
	TCPBuffer            TypeBytes       `json:"tcpBuffer"`
	PreferIP             TypePreferIP    `json:"preferIp"`
	DomainFrontingPort   TypePort        `json:"domainFrontingPort"`
	TolerateTimeSkewness TypeDuration    `json:"tolerateTimeSkewness"`
	Concurrency          TypeConcurrency `json:"concurrency"`
	Defense              struct {
		AntiReplay struct {
			Enabled   TypeBool      `json:"enabled"`
			MaxSize   TypeBytes     `json:"maxSize"`
			ErrorRate TypeErrorRate `json:"errorRate"`
		} `json:"antiReplay"`
		Blocklist struct {
			Enabled             TypeBool           `json:"enabled"`
			DownloadConcurrency TypeConcurrency    `json:"downloadConcurrency"`
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
		DOHIP   TypeIP         `json:"dohIp"`
		Proxies []TypeProxyURL `json:"proxies"`
	} `json:"network"`
	Stats struct {
		StatsD struct {
			Enabled      TypeBool            `json:"enabled"`
			Address      TypeHostPort        `json:"address"`
			MetricPrefix TypeMetricPrefix    `json:"metricPrefix"`
			TagFormat    TypeStatsdTagFormat `json:"tagFormat"`
		} `json:"statsd"`
		Prometheus struct {
			Enabled      TypeBool         `json:"enabled"`
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

    if c.BindTo.Get("") == "" {
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
