package config

import (
	"fmt"
	"strings"
	"time"
)

type TypeDuration struct {
	value time.Duration
}

func (c *TypeDuration) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	dur, err := time.ParseDuration(strings.ToLower(string(data)))
	if err != nil {
		return fmt.Errorf("incorrect duration: %w", err)
	}

	if dur < 0 {
		return fmt.Errorf("%s should be positive duration", dur)
	}

	c.value = dur

	return nil
}

func (c TypeDuration) MarshalText() ([]byte, error) { // nolint: unparam
	return []byte(c.value.String()), nil
}

func (c TypeDuration) String() string {
	return c.value.String()
}

func (c TypeDuration) Value(defaultValue time.Duration) time.Duration {
	if c.value == 0 {
		return defaultValue
	}

	return c.value
}
