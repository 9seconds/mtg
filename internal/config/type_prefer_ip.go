package config

import (
	"fmt"
	"strings"
)

const (
	// TypePreferIPPreferIPv4 states that you prefer to use IPv4 addresses
	// but IPv6 is also possible.
	TypePreferIPPreferIPv4 = "prefer-ipv4"

	// TypePreferIPPreferIPv6 states that you prefer to use IPv6 addresses
	// but IPv4 is also possible.
	TypePreferIPPreferIPv6 = "prefer-ipv6"

	// TypePreferOnlyIPv4 states that you prefer to use IPv4 addresses
	// only.
	TypePreferOnlyIPv4 = "only-ipv4"

	// TypePreferOnlyIPv6 states that you prefer to use IPv6 addresses
	// only.
	TypePreferOnlyIPv6 = "only-ipv6"
)

type TypePreferIP struct {
	Value string
}

func (t *TypePreferIP) Set(value string) error {
	value = strings.ToLower(value)

	switch value {
	case TypePreferIPPreferIPv4, TypePreferIPPreferIPv6,
		TypePreferOnlyIPv4, TypePreferOnlyIPv6:
		t.Value = value

		return nil
	default:
		return fmt.Errorf("unsupported ip preference: %s", value)
	}
}

func (t *TypePreferIP) Get(defaultValue string) string {
	if t.Value == "" {
		return defaultValue
	}

	return t.Value
}

func (t *TypePreferIP) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypePreferIP) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypePreferIP) String() string {
	return t.Value
}
