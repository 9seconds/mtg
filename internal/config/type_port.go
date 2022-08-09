package config

import (
	"fmt"
	"strconv"
)

type TypePort struct {
	Value uint
}

func (t *TypePort) Set(value string) error {
	portValue, err := strconv.ParseUint(value, 10, 16) //nolint: gomnd
	if err != nil {
		return fmt.Errorf("incorrect port number (%v): %w", value, err)
	}

	if portValue == 0 {
		return fmt.Errorf("incorrect port number (%s)", value)
	}

	t.Value = uint(portValue)

	return nil
}

func (t TypePort) Get(defaultValue uint) uint {
	if t.Value == 0 {
		return defaultValue
	}

	return t.Value
}

func (t *TypePort) UnmarshalJSON(data []byte) error {
	return t.Set(string(data))
}

func (t TypePort) MarshalJSON() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypePort) String() string {
	return strconv.Itoa(int(t.Value))
}
