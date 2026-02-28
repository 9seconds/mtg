package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/url"

	"github.com/9seconds/mtg/v2/mtglib"
)

type Optional struct {
	Enabled TypeBool `json:"enabled"`
}

type ListConfig struct {
	Optional

	DownloadConcurrency TypeConcurrency    `json:"downloadConcurrency"`
	URLs                []TypeBlocklistURI `json:"urls"`
	UpdateEach          TypeDuration       `json:"updateEach"`
}

type Config struct {
	Debug                       TypeBool        `json:"debug"`
	AllowFallbackOnUnknownDC    TypeBool        `json:"allowFallbackOnUnknownDc"`
	Secret                      mtglib.Secret   `json:"secret"`
	BindTo                      TypeHostPort    `json:"bindTo"`
	ProxyProtocolListener       TypeBool        `json:"proxyProtocolListener"`
	PreferIP                    TypePreferIP    `json:"preferIp"`
	AutoUpdate                  TypeBool        `json:"autoUpdate"`
	DomainFrontingPort          TypePort        `json:"domainFrontingPort"`
	DomainFrontingIP            TypeIP          `json:"domainFrontingIp"`
	DomainFrontingProxyProtocol TypeBool        `json:"domainFrontingProxyProtocol"`
	TolerateTimeSkewness        TypeDuration    `json:"tolerateTimeSkewness"`
	Concurrency                 TypeConcurrency `json:"concurrency"`
	DomainFronting              struct {
		IP            TypeIP   `json:"ip"`
		Port          TypePort `json:"port"`
		ProxyProtocol TypeBool `json:"proxyProtocol"`
	} `json:"domainFronting"`
	Defense struct {
		AntiReplay struct {
			Optional

			MaxSize   TypeBytes     `json:"maxSize"`
			ErrorRate TypeErrorRate `json:"errorRate"`
		} `json:"antiReplay"`
		Blocklist ListConfig `json:"blocklist"`
		Allowlist ListConfig `json:"allowlist"`
	} `json:"defense"`
	Network struct {
		Timeout struct {
			TCP  TypeDuration `json:"tcp"`
			HTTP TypeDuration `json:"http"`
			Idle TypeDuration `json:"idle"`
		} `json:"timeout"`
		DOHIP   TypeIP         `json:"dohIp"`
		DNS     TypeDNSURI     `json:"dns"`
		Proxies []TypeProxyURL `json:"proxies"`
	} `json:"network"`
	Stats struct {
		StatsD struct {
			Optional

			Address      TypeHostPort        `json:"address"`
			MetricPrefix TypeMetricPrefix    `json:"metricPrefix"`
			TagFormat    TypeStatsdTagFormat `json:"tagFormat"`
		} `json:"statsd"`
		Prometheus struct {
			Optional

			BindTo       TypeHostPort     `json:"bindTo"`
			HTTPPath     TypeHTTPPath     `json:"httpPath"`
			MetricPrefix TypeMetricPrefix `json:"metricPrefix"`
		} `json:"prometheus"`
	} `json:"stats"`
}

func (c *Config) GetDNS() *url.URL {
	var dohURL *url.URL

	if dohIP := c.Network.DOHIP.Get(nil); dohIP != nil {
		dohURL, _ = url.Parse("https://" + dohIP.String())
	}

	return c.Network.DNS.Get(dohURL)
}

func (c *Config) GetDomainFrontingPort(defaultValue uint) uint {
	if port := c.DomainFronting.Port.Get(0); port != 0 {
		return port
	}
	return c.DomainFrontingPort.Get(defaultValue)
}

func (c *Config) GetDomainFrontingIP(defaultValue net.IP) string {
	if ip := c.DomainFronting.IP.Get(nil); ip != nil {
		return ip.String()
	}
	if ip := c.DomainFrontingIP.Get(defaultValue); ip != nil {
		return ip.String()
	}
	return ""
}

func (c *Config) GetDomainFrontingProxyProtocol(defaultValue bool) bool {
	return c.DomainFronting.ProxyProtocol.Get(false) || c.DomainFrontingProxyProtocol.Get(defaultValue)
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
