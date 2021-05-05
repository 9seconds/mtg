package config

import (
	"fmt"
	"net"
	"strconv"
)

type TypeHostPort struct {
	host TypeIP
	port TypePort
}

func (c *TypeHostPort) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	text := string(data)

	host, port, err := net.SplitHostPort(text)
	if err != nil {
		return fmt.Errorf("incorrect host:port syntax: %w", err)
	}

	if port == "" {
		return fmt.Errorf("port in %s host:port pair cannot be empty", text)
	}

	if err := c.port.UnmarshalJSON([]byte(port)); err != nil {
		return fmt.Errorf("incorrect port in host:port: %w", err)
	}

	if err := c.host.UnmarshalText([]byte(host)); err != nil {
		return fmt.Errorf("incorrect host: %w", err)
	}

	return nil
}

func (c TypeHostPort) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c TypeHostPort) String() string {
	return c.Value(net.IP{}, 0)
}

func (c TypeHostPort) HostValue(defaultValue net.IP) net.IP {
	return c.host.Value(defaultValue)
}

func (c TypeHostPort) PortValue(defaultValue uint) uint {
	return c.port.Value(defaultValue)
}

func (c TypeHostPort) Value(defaultHostValue net.IP, defaultPortValue uint) string {
	host := c.HostValue(defaultHostValue)
	port := c.PortValue(defaultPortValue)

	hostStr := ""
	if len(host) > 0 {
		hostStr = host.String()
	}

	return net.JoinHostPort(hostStr, strconv.Itoa(int(port)))
}
