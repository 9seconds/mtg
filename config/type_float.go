package config

import (
	"fmt"
	"strconv"
)

type TypeFloat struct {
	value float64
}

func (c *TypeFloat) UnmarshalJSON(data []byte) error {
	value, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return fmt.Errorf("incorrect float value: %w", err)
	}

	if value < 0 {
		return fmt.Errorf("%f should be positive", value)
	}

	c.value = value

	return nil
}

func (c *TypeFloat) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c TypeFloat) String() string {
	return strconv.FormatFloat(c.value, 'f', -1, 64)
}

func (c TypeFloat) Value(defaultValue float64) float64 {
	if c.value < 0.00001 {
		return defaultValue
	}

	return c.value
}
