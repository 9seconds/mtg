package config

import (
	"fmt"
	"strings"
)

const (
	// TypeStatsdTagFormatInfluxdb defines a tag format compatible with
	// InfluxDB.
	TypeStatsdTagFormatInfluxdb = "influxdb"

	// TypeStatsdTagFormatDatadog defines a tag format compatible with
	// DataDog.
	TypeStatsdTagFormatDatadog = "datadog"

	// TypeStatsdTagFormatGraphite defines a tag format compatible with
	// Graphite.
	TypeStatsdTagFormatGraphite = "graphite"
)

type TypeStatsdTagFormat struct {
	Value string
}

func (t *TypeStatsdTagFormat) Set(value string) error {
	lowercasedValue := strings.ToLower(value)

	switch lowercasedValue {
	case TypeStatsdTagFormatDatadog, TypeStatsdTagFormatInfluxdb,
		TypeStatsdTagFormatGraphite:
		t.Value = lowercasedValue

		return nil
	default:
		return fmt.Errorf("unknown tag format %s", value)
	}
}

func (t TypeStatsdTagFormat) Get(defaultValue string) string {
	if t.Value == "" {
		return defaultValue
	}

	return t.Value
}

func (t *TypeStatsdTagFormat) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t *TypeStatsdTagFormat) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t *TypeStatsdTagFormat) String() string {
	return t.Value
}
