package config

import (
	"fmt"
	"strings"

	"github.com/alecthomas/units"
)

type TypeBytes struct {
	value uint
}

func (c *TypeBytes) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	value, err := units.ParseStrictBytes(strings.ToUpper(string(data)))
	if err != nil {
		return fmt.Errorf("incorrect bytes value: %w", err)
	}

	if value < 0 {
		return fmt.Errorf("%d should be positive number", value)
	}

	c.value = uint(value)

	return nil
}

func (c TypeBytes) MarshalText() ([]byte, error) { // nolint: unparam
	return []byte(c.String()), nil
}

func (c TypeBytes) String() string {
	return units.ToString(int64(c.value), 1024, "ib", "b")
}

func (c TypeBytes) Value(defaultValue uint) uint {
	if c.value == 0 {
		return defaultValue
	}

	return c.value
}
