package config

import (
	"fmt"
	"regexp"
)

type TypeMetricPrefix struct {
	Value string
}

func (t *TypeMetricPrefix) Set(value string) error {
	if ok, err := regexp.MatchString("^[a-z0-9]+$", value); !ok || err != nil {
		return fmt.Errorf("incorrect metric prefix %s: %w", value, err)
	}

	t.Value = value

	return nil
}

func (t TypeMetricPrefix) Get(defaultValue string) string {
	if t.Value == "" {
		return defaultValue
	}

	return t.Value
}

func (t *TypeMetricPrefix) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeMetricPrefix) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeMetricPrefix) String() string {
	return t.Value
}
