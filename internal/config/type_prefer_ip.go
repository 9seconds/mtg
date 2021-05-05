package config

import (
	"fmt"
	"strings"
)

const (
	TypePreferIPPreferIPv4 = "prefer-ipv4"
	TypePreferIPPreferIPv6 = "prefer-ipv6"
	TypePreferOnlyIPv4     = "only-ipv4"
	TypePreferOnlyIPv6     = "only-ipv6"
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
	case TypePreferIPPreferIPv4, TypePreferIPPreferIPv6, TypePreferOnlyIPv4, TypePreferOnlyIPv6:
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
