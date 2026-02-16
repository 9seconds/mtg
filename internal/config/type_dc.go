package config

import (
	"fmt"
	"strconv"
)

type TypeDC struct {
	Value int
}

func (t *TypeDC) Set(value string) error {
	parsed, err := strconv.ParseInt(value, 10, 16)
	if err != nil {
		return fmt.Errorf("cannot parse dc: %w", err)
	}

	if parsed < 0 {
		parsed = -parsed
	}

	t.Value = int(parsed)

	return nil
}

func (t *TypeDC) UnmarshalJSON(data []byte) error {
	return t.Set(string(data))
}

func (t TypeDC) MarshalJSON() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeDC) String() string {
	return strconv.Itoa(t.Value)
}

func (t TypeDC) Get() int {
	return t.Value
}
