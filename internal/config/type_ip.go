package config

import (
	"fmt"
	"net"
)

type TypeIP struct {
	Value net.IP
}

func (t *TypeIP) Set(value string) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return fmt.Errorf("incorret ip %s", value)
	}

	t.Value = ip

	return nil
}

func (t *TypeIP) Get(defaultValue net.IP) net.IP {
	if len(t.Value) == 0 {
		return defaultValue
	}

	return t.Value
}

func (t *TypeIP) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeIP) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeIP) String() string {
	if len(t.Value) == 0 {
		return ""
	}

	return t.Value.String()
}
