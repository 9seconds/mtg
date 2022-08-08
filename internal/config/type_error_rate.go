package config

import (
	"fmt"
	"strconv"
)

const typeErrorRateIgnoreLess = 1e-8

type TypeErrorRate struct {
	Value float64
}

func (t *TypeErrorRate) Set(value string) error {
	parsedValue, err := strconv.ParseFloat(value, 64) //nolint: gomnd
	if err != nil {
		return fmt.Errorf("value is not a float (%s): %w", value, err)
	}

	if parsedValue <= 0.0 || parsedValue >= 100.0 {
		return fmt.Errorf("value should be 0 < x < 100 (%s)", value)
	}

	t.Value = parsedValue

	return nil
}

func (t TypeErrorRate) Get(defaultValue float64) float64 {
	if t.Value < typeErrorRateIgnoreLess {
		return defaultValue
	}

	return t.Value
}

func (t *TypeErrorRate) UnmarshalJSON(data []byte) error {
	return t.Set(string(data))
}

func (t TypeErrorRate) MarshalJSON() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeErrorRate) String() string {
	return strconv.FormatFloat(t.Value, 'f', -1, 64) //nolint: gomnd
}
