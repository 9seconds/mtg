package config

import (
	"fmt"
	"net"
)

type TypeIP struct {
	value net.IP
}

func (c *TypeIP) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	ip := net.ParseIP(string(data))
	if ip == nil {
		return fmt.Errorf("incorrect ip address: %s", string(data))
	}

	c.value = ip

	return nil
}

func (c *TypeIP) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c TypeIP) String() string {
	if len(c.value) > 0 {
		return c.value.String()
	}

	return ""
}

func (c TypeIP) Value(defaultValue net.IP) net.IP {
	if c.value == nil {
		return defaultValue
	}

	return c.value
}
