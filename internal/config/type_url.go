package config

import (
	"fmt"
	"net"
	"net/url"
)

type TypeURL struct {
	value *url.URL
}

func (c *TypeURL) UnmarshalText(data []byte) error { // nolint: cyclop
	if len(data) == 0 {
		return nil
	}

	value, err := url.Parse(string(data))
	if err != nil {
		return fmt.Errorf("incorrect URL: %w", err)
	}

	switch value.Scheme {
	case "http", "https", "socks5":
	case "":
		return fmt.Errorf("url %s has to have a schema", value)
	default:
		return fmt.Errorf("unsupported schema %s", value.Scheme)
	}

	if value.Host == "" {
		return fmt.Errorf("url %s has to have a host", value)
	}

	if _, _, err := net.SplitHostPort(value.Host); err != nil {
		switch value.Scheme {
		case "http":
			value.Host = net.JoinHostPort(value.Host, "80")
		case "https":
			value.Host = net.JoinHostPort(value.Host, "443")
		case "socks5":
			value.Host = net.JoinHostPort(value.Host, "1080")
		default:
			return fmt.Errorf("cannot set a default port for %s", value)
		}
	}

	c.value = value

	return nil
}

func (c *TypeURL) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c TypeURL) String() string {
	if c.value == nil {
		return ""
	}

	return c.value.String()
}

func (c TypeURL) Value(defaultValue *url.URL) *url.URL {
	if c.value == nil {
		return defaultValue
	}

	return c.value
}
