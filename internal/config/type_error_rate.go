package config

import (
	"fmt"
	"strconv"
)

const typeErrorRateIgnoreLess = 1e-8

type TypeErrorRate struct {
	value float64
}

func (c *TypeErrorRate) UnmarshalJSON(data []byte) error {
	value, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return fmt.Errorf("incorrect float value: %w", err)
	}

	if value <= 0 || value >= 100 {
		return fmt.Errorf("%f should be 0 < x < 100", value)
	}

	c.value = value

	return nil
}

func (c *TypeErrorRate) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c TypeErrorRate) String() string {
	return strconv.FormatFloat(c.value, 'f', -1, 64)
}

func (c TypeErrorRate) Value(defaultValue float64) float64 {
	if c.value < typeErrorRateIgnoreLess {
		return defaultValue
	}

	return c.value
}
