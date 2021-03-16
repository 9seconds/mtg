package config

import (
	"fmt"
	"strings"
)

const (
	TypeStatsdTagFormatInfluxdb = "influxdb"
	TypeStatsdTagFormatDatadog  = "datadog"
	TypeStatsdTagFormatGraphite = "graphite"
)

type TypeStatsdTagFormat struct {
	value string
}

func (c *TypeStatsdTagFormat) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	text := strings.ToLower(string(data))

	switch text {
	case TypeStatsdTagFormatInfluxdb, TypeStatsdTagFormatDatadog, TypeStatsdTagFormatGraphite:
		c.value = text
	default:
		return fmt.Errorf("incorrect tag format value: %s", string(data))
	}

	return nil
}

func (c TypeStatsdTagFormat) MarshalText() ([]byte, error) {
	return []byte(c.value), nil
}

func (c *TypeStatsdTagFormat) String() string {
	return c.value
}

func (c *TypeStatsdTagFormat) Value(defaultValue string) string {
	if c.value == "" {
		return defaultValue
	}

	return c.value
}
