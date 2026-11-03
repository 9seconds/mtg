package config

import (
	"fmt"
	"net"
	"strings"
)

// TypeHost is a non-empty string that is either a literal IP address
// (IPv4 or IPv6) or a hostname suitable for DNS resolution. It does not
// include a port — the port belongs in a separate field.
type TypeHost struct {
	Value string
}

func (t *TypeHost) Set(value string) error {
	if value == "" {
		return fmt.Errorf("host cannot be empty")
	}

	if net.ParseIP(value) != nil {
		t.Value = value

		return nil
	}

	if strings.ContainsAny(value, " \t\n/?#") {
		return fmt.Errorf("incorrect host %q", value)
	}

	// At this point value is not a parsed IP (IPv6 literals returned
	// above), so any remaining colon indicates a host:port form, which
	// belongs in a separate field.
	if strings.Contains(value, ":") {
		return fmt.Errorf("host must not contain a port: %q", value)
	}

	t.Value = value

	return nil
}

func (t TypeHost) Get(defaultValue string) string {
	if t.Value == "" {
		return defaultValue
	}

	return t.Value
}

func (t *TypeHost) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeHost) MarshalText() ([]byte, error) {
	return []byte(t.Value), nil
}

func (t TypeHost) String() string {
	return t.Value
}
