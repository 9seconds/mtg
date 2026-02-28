package config

import (
	"fmt"
	"net"
	"net/url"
)

type TypeDNSURI struct {
	Value *url.URL
}

func (t *TypeDNSURI) Set(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("value is not URI: %w", err)
	}

	if parsed.Host == "" {
		parsed.Host = parsed.Path
		parsed.Path = ""
		parsed.Scheme = "udp"
	}

	switch parsed.Scheme {
	case "https", "tls":
	case "udp":
		if ip := net.ParseIP(parsed.Hostname()); ip == nil {
			return fmt.Errorf("simple DNS must IP address: %s", parsed.Hostname())
		}
	default:
		return fmt.Errorf("unsupported DNS type %s", parsed.Scheme)
	}

	if parsed.Scheme != "https" && parsed.Path != "" {
		return fmt.Errorf("path is supported only for DoH: %s", parsed)
	}

	if parsed.User != nil {
		return fmt.Errorf("used info is not supported: %s", parsed.User.String())
	}

	t.Value = parsed

	return nil
}

func (t *TypeDNSURI) Get(defaultValue *url.URL) *url.URL {
	if t.Value != nil {
		return t.Value
	}

	return defaultValue
}

func (t *TypeDNSURI) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeDNSURI) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeDNSURI) String() string {
	if t.Value == nil {
		return ""
	}
	return t.Value.String()
}
