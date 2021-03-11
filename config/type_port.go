package config

import (
	"fmt"
	"strconv"
)

type TypePort struct {
	value uint
}

func (c *TypePort) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	intValue, err := strconv.ParseUint(string(data), 10, 16)
	if err != nil {
		return fmt.Errorf("port number is not a number: %w", err)
	}

	if intValue == 0 || intValue > 65536 {
		return fmt.Errorf("port number should be 0 < portNo < 65536: %d", intValue)
	}

	c.value = uint(intValue)

	return nil
}

func (c *TypePort) MarshalJSON() ([]byte, error) { // nolint: unparam
    return []byte(c.String()), nil
}

func (c TypePort) String() string {
	return strconv.Itoa(int(c.value))
}

func (c TypePort) Value(defaultValue uint) uint {
	if c.value == 0 {
		return defaultValue
	}

	return c.value
}
