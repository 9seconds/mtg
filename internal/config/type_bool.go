package config

import (
	"fmt"
	"strconv"
)

type TypeBool struct {
	Value bool
}

func (t *TypeBool) Set(data string) error {
	parsed, err := strconv.ParseBool(data)
	if err != nil {
		return fmt.Errorf("incorrect bool value: %s", data)
	}

	t.Value = parsed

	return nil
}

func (t TypeBool) Get(defaultValue bool) bool {
	return t.Value || defaultValue
}

func (t *TypeBool) UnmarshalJSON(data []byte) error {
	return t.Set(string(data))
}

func (t TypeBool) MarshalJSON() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeBool) String() string {
	return strconv.FormatBool(t.Value)
}
