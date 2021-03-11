package config

import "strings"

type TypeHTTPPath struct {
	value string
}

func (c *TypeHTTPPath) UnmarshalText(data []byte) error { // nolint: unparam
	if len(data) > 0 {
		c.value = "/" + strings.Trim(string(data), "/")
	}

	return nil
}

func (c TypeHTTPPath) MarshalText() ([]byte, error) { // nolint: unparam
	return []byte(c.String()), nil
}

func (c TypeHTTPPath) String() string {
	return c.value
}

func (c TypeHTTPPath) Value(defaultValue string) string {
	if c.value == "" {
		return defaultValue
	}

	return c.value
}
