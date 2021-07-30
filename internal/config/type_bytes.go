package config

import (
	"fmt"
	"strings"

	"github.com/alecthomas/units"
)

var typeBytesStringCleaner = strings.NewReplacer(" ", "", "\t", "", "IB", "iB")

type TypeBytes struct {
	Value units.Base2Bytes
}

func (t *TypeBytes) Set(value string) error {
	normalizedValue := typeBytesStringCleaner.Replace(strings.ToUpper(value))

	parsedValue, err := units.ParseBase2Bytes(normalizedValue)
	if err != nil {
		return fmt.Errorf("incorrect bytes value (%v): %w", value, err)
	}

	if parsedValue < 0 {
		return fmt.Errorf("bytes should be positive (%s)", value)
	}

	t.Value = parsedValue

	return nil
}

func (t TypeBytes) Get(defaultValue uint) uint {
	if t.Value == 0 {
		return defaultValue
	}

	return uint(t.Value)
}

func (t *TypeBytes) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeBytes) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeBytes) String() string {
	if t.Value == 0 {
		return ""
	}

	return strings.ToLower(t.Value.String())
}
