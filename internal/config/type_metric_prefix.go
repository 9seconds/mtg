package config

import (
	"fmt"
	"regexp"
)

type TypeMetricPrefix struct {
	value string
}

func (c *TypeMetricPrefix) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	prefix := string(data)
	if ok, err := regexp.MatchString("^[a-z0-9]+$", prefix); !ok || err != nil {
		return fmt.Errorf("incorrect metric prefix: %s", prefix)
	}

	c.value = prefix

	return nil
}

func (c TypeMetricPrefix) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c TypeMetricPrefix) String() string {
	return c.value
}

func (c TypeMetricPrefix) Value(defaultValue string) string {
	if c.value == "" {
		return defaultValue
	}

	return c.value
}
