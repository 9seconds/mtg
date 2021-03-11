package config

import (
	"fmt"
	"strings"
)

type TypePreferIP struct {
	value string
}

func (c *TypePreferIP) UnmarshalText(data []byte) error {
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

func (c TypePreferIP) MarshalText() ([]byte, error) {
	return []byte(c.value), nil
}

func (c *TypePreferIP) String() string {
	return c.value
}

func (c *TypePreferIP) Value(defaultValue string) string {
	if c.value == "" {
		return defaultValue
	}

	return c.value
}
