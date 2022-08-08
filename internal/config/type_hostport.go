package config

import (
	"fmt"
	"net"
	"strconv"
)

type TypeHostPort struct {
	Value string
	Host  string
	Port  uint
}

func (t *TypeHostPort) Set(value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("incorrect host:port value (%v): %w", value, err)
	}

	portValue, err := strconv.ParseUint(port, 10, 16) //nolint: gomnd
	if err != nil {
		return fmt.Errorf("incorrect port number (%v): %w", value, err)
	}

	if portValue == 0 {
		return fmt.Errorf("incorrect port number (%s)", value)
	}

	if host == "" {
		return fmt.Errorf("empty host: %s", value)
	}

	if net.ParseIP(host) == nil {
		return fmt.Errorf("host is not an IP address: %s", value)
	}

	t.Value = net.JoinHostPort(host, port)
	t.Port = uint(portValue)
	t.Host = host

	return nil
}

func (t TypeHostPort) Get(defaultValue string) string {
	if t.Value == "" {
		return defaultValue
	}

	return t.Value
}

func (t *TypeHostPort) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeHostPort) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeHostPort) String() string {
	return t.Value
}
