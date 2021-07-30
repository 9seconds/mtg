package config

import (
	"fmt"
	"net"
	"net/url"
)

const typeProxyURLDefaultSOCKS5Port = "1080"

type TypeProxyURL struct {
	Value *url.URL
}

func (t *TypeProxyURL) Set(value string) error {
	parsedURL, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("value is not corect URL (%s): %w", value, err)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("url has to have a schema: %s", value)
	}

	if parsedURL.Scheme != "socks5" {
		return fmt.Errorf("unsupported schema: %s", parsedURL.Scheme)
	}

	if _, _, err := net.SplitHostPort(parsedURL.Host); err != nil {
		parsedURL.Host = net.JoinHostPort(parsedURL.Host,
			typeProxyURLDefaultSOCKS5Port)
	}

	t.Value = parsedURL

	return nil
}

func (t *TypeProxyURL) Get(defaultValue *url.URL) *url.URL {
	if t.Value == nil {
		return defaultValue
	}

	return t.Value
}

func (t *TypeProxyURL) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeProxyURL) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeProxyURL) String() string {
	if t.Value == nil {
		return ""
	}

	return t.Value.String()
}
