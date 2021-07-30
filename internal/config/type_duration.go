package config

import (
	"fmt"
	"strings"
	"time"
)

var typeDurationStringCleaner = strings.NewReplacer(" ", "", "\t", "")

type TypeDuration struct {
	Value time.Duration
}

func (t *TypeDuration) Set(value string) error {
	parsedValue, err := time.ParseDuration(
		typeDurationStringCleaner.Replace(strings.ToLower(value)))
	if err != nil {
		return fmt.Errorf("incorrect duration (%s): %w", value, err)
	}

	if parsedValue < 0 {
		return fmt.Errorf("duration has to be a positive: %s", value)
	}

	t.Value = parsedValue

	return nil
}

func (t TypeDuration) Get(defaultValue time.Duration) time.Duration {
	if t.Value == 0 {
		return defaultValue
	}

	return t.Value
}

func (t *TypeDuration) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeDuration) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeDuration) String() string {
	if t.Value == 0 {
		return ""
	}

	return t.Value.String()
}
