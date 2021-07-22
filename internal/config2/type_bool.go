package config2

import (
	"fmt"
	"strconv"
	"strings"
)

type TypeBool struct {
	Value bool
}

func (t *TypeBool) Set(data string) error {
	switch strings.ToLower(data) {
	case "1", "y", "yes", "enabled", "true":
		t.Value = true
	case "0", "n", "no", "disabled", "false":
		t.Value = false
	default:
		return fmt.Errorf("incorrect bool value %s", data)
	}

	return nil
}

func (t TypeBool) Get(defaultValue bool) bool {
	return t.Value || defaultValue
}

func (t *TypeBool) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeBool) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeBool) String() string {
	return strconv.FormatBool(t.Value)
}
