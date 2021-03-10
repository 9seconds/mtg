package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/alecthomas/units"
	"github.com/pelletier/go-toml"
)

type configTypeHostPort struct {
	host configTypeIP
	port configTypePort
}

func (c *configTypeHostPort) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	host, port, err := net.SplitHostPort(string(data))
	if err != nil {
		return fmt.Errorf("incorrect host:port syntax: %w", err)
	}

	if err := c.port.UnmarshalJSON([]byte(port)); err != nil {
		return fmt.Errorf("incorrect port in host:port: %w", err)
	}

	if err := c.host.UnmarshalText([]byte(host)); err != nil {
		return fmt.Errorf("incorrect host: %w", err)
	}

	return nil
}

func (c configTypeHostPort) String() string {
	return c.Value(net.IP{}, 0)
}

func (c configTypeHostPort) Value(defaultHostValue net.IP, defaultPortValue uint) string {
	return net.JoinHostPort(c.host.Value(defaultHostValue).String(),
		strconv.Itoa(int(c.port.Value(defaultPortValue))))
}

type configTypePort struct {
	value uint
}

func (c *configTypePort) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	intValue, err := strconv.ParseUint(string(data), 10, 16)
	if err != nil {
		return fmt.Errorf("port number is not a number: %w", err)
	}

	if intValue == 0 || intValue > 65536 {
		return fmt.Errorf("port number should be 0 < portNo < 65536: %d", intValue)
	}

	c.value = uint(intValue)

	return nil
}

func (c configTypePort) String() string {
	return strconv.Itoa(int(c.value))
}

func (c configTypePort) Value(defaultValue uint) uint {
	if c.value == 0 {
		return defaultValue
	}

	return c.value
}

type configTypeBytes struct {
	value uint
}

func (c *configTypeBytes) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	value, err := units.ParseStrictBytes(strings.ToUpper(string(data)))
	if err != nil {
		return fmt.Errorf("incorrect bytes value: %w", err)
	}

	if value < 0 {
		return fmt.Errorf("%d should be positive number", value)
	}

	c.value = uint(value)

	return nil
}

func (c configTypeBytes) String() string {
	return units.ToString(int64(c.value), 1024, "ib", "b")
}

func (c configTypeBytes) Value(defaultValue uint) uint {
	if c.value == 0 {
		return defaultValue
	}

	return c.value
}

type configTypePreferIP struct {
	value string
}

func (c *configTypePreferIP) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	text := strings.ToLower(string(data))

	switch text {
	case "prefer-ipv4", "prefer-ipv6", "only-ipv4", "only-ipv6":
		c.value = text
	default:
		return fmt.Errorf("incorrect prefer-ip value: %s", string(data))
	}

	return nil
}

func (c *configTypePreferIP) String() string {
	return c.value
}

func (c *configTypePreferIP) Value(defaultValue string) string {
	if c.value == "" {
		return defaultValue
	}

	return c.value
}

type configTypeDuration struct {
	value time.Duration
}

func (c *configTypeDuration) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	dur, err := time.ParseDuration(strings.ToLower(string(data)))
	if err != nil {
		return fmt.Errorf("incorrect duration: %w", err)
	}

	if dur < 0 {
		return fmt.Errorf("%s should be positive duration", dur)
	}

	c.value = dur

	return nil
}

func (c configTypeDuration) String() string {
	return c.value.String()
}

func (c configTypeDuration) Value(defaultValue time.Duration) time.Duration {
	if c.value == 0 {
		return defaultValue
	}

	return c.value
}

type configTypeFloat struct {
	value float64
}

func (c *configTypeFloat) UnmarshalJSON(data []byte) error {
	value, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return fmt.Errorf("incorrect float value: %w", err)
	}

	if value < 0 {
		return fmt.Errorf("%f should be positive", value)
	}

	c.value = value

	return nil
}

func (c configTypeFloat) String() string {
	return strconv.FormatFloat(c.value, 'f', -1, 64)
}

func (c configTypeFloat) Value(defaultValue float64) float64 {
	if c.value < 0.00001 {
		return defaultValue
	}

	return c.value
}

type configTypeIP struct {
	value net.IP
}

func (c *configTypeIP) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	ip := net.ParseIP(string(data))
	if ip == nil {
		return fmt.Errorf("incorrect ip address: %s", string(data))
	}

	c.value = ip

	return nil
}

func (c configTypeIP) String() string {
	return c.value.String()
}

func (c configTypeIP) Value(defaultValue net.IP) net.IP {
	if c.value == nil {
		return defaultValue
	}

	return c.value
}

type configTypeURL struct {
	value *url.URL
}

func (c *configTypeURL) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	value, err := url.Parse(string(data))
	if err != nil {
		return fmt.Errorf("incorrect URL: %w", err)
	}

	c.value = value

	return nil
}

func (c configTypeURL) String() string {
	if c.value == nil {
		return ""
	}

	return c.value.String()
}

func (c configTypeURL) Value(defaultValue *url.URL) *url.URL {
	if c.value == nil {
		return defaultValue
	}

	return c.value
}

type configTypeMetricPrefix struct {
	value string
}

func (c *configTypeMetricPrefix) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	prefix := string(data)

	if ok, err := regexp.MatchString("^[a-z0-9]+$", prefix); !ok || err != nil {
		return fmt.Errorf("incorrect metric prefix: %s", prefix)
	}

	c.value = prefix

	return nil
}

func (c configTypeMetricPrefix) String() string {
	return c.value
}

func (c configTypeMetricPrefix) Value(defaultValue string) string {
	if c.value == "" {
		return defaultValue
	}

	return c.value
}

type configTypeHTTPPath struct {
	value string
}

func (c *configTypeHTTPPath) UnmarshalText(data []byte) error { // nolint: unparam
	if len(data) > 0 {
		c.value = "/" + strings.Trim(string(data), "/")
	}

	return nil
}

func (c configTypeHTTPPath) String() string {
	return c.value
}

func (c configTypeHTTPPath) Value(defaultValue string) string {
	if c.value == "" {
		return defaultValue
	}

	return c.value
}

type config struct {
	Debug     bool               `json:"debug"`
	Secret    mtglib.Secret      `json:"secret"`
	BindTo    configTypeHostPort `json:"bind-to"`
	TCPBuffer configTypeBytes    `json:"tcp-buffer"`
	PreferIP  configTypePreferIP `json:"prefer-ip"`
	CloakPort configTypePort     `json:"cloak-port"`
	Probes    struct {
		Time struct {
			Enabled       bool               `json:"enabled"`
			AllowSkewness configTypeDuration `json:"allow-skewness"`
		} `json:"time"`
		AntiReplay struct {
			Enabled   bool            `json:"enabled"`
			MaxSize   configTypeBytes `json:"max-size"`
			ErrorRate configTypeFloat `json:"error-rate"`
		} `json:"anti-replay"`
	} `json:"probes"`
	Network struct {
		PublicIP struct {
			IPv4 configTypeIP `json:"ipv4"`
			IPv6 configTypeIP `json:"ipv6"`
		} `json:"public-ip"`
		DOHIP   configTypeIP    `json:"doh-ip"`
		Proxies []configTypeURL `json:"proxies"`
	} `json:"network"`
	Stats struct {
		StatsD struct {
			Enabled      bool                   `json:"enabled"`
			Address      configTypeHostPort     `json:"address"`
			MetricPrefix configTypeMetricPrefix `json:"metric-prefix"`
		} `json:"statsd"`
		Prometheus struct {
			Enabled      bool                   `json:"enabled"`
			BindTo       configTypeHostPort     `json:"bind-to"`
			HTTPPath     configTypeHTTPPath     `json:"http-path"`
			MetricPrefix configTypeMetricPrefix `json:"metric-prefix"`
		} `json:"prometheus"`
	} `json:"stats"`
}

func (c *config) Validate() error {
	if len(c.Secret.Key) == 0 || c.Secret.Host == "" {
		return fmt.Errorf("incorrect secret %s", c.Secret.String())
	}

	return nil
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

	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("cannot validate config: %w", err)
	}

	return conf, nil
}
