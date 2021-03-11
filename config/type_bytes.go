package config

import (
	"fmt"
	"strings"

	"github.com/alecthomas/units"
)

type TypeBytes struct {
	value units.Base2Bytes
}

func (c *TypeBytes) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	normalizedData := strings.ToUpper(string(data))
	normalizedData = strings.ReplaceAll(normalizedData, "IB", "iB")

	value, err := units.ParseBase2Bytes(normalizedData)
	if err != nil {
		return fmt.Errorf("incorrect bytes value: %w", err)
	}

	if value < 0 {
		return fmt.Errorf("%d should be positive number", value)
	}

	c.value = value

	return nil
}

func (c TypeBytes) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c TypeBytes) String() string {
	return strings.ToLower(c.value.String())
}

func (c TypeBytes) Value(defaultValue uint) uint {
	if c.value == 0 {
		return defaultValue
	}

	return uint(c.value)
}
