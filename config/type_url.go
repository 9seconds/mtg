package config

import (
	"fmt"
	"net/url"
)

type TypeURL struct {
	value *url.URL
}

func (c *TypeURL) UnmarshalText(data []byte) error {
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
