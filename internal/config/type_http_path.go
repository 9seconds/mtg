package config

import "strings"

type TypeHTTPPath struct {
	Value string
}

func (t *TypeHTTPPath) Set(value string) error {
	t.Value = "/" + strings.Trim(value, "/")

	return nil
}

func (t TypeHTTPPath) Get(defaultValue string) string {
	if t.Value == "" {
		return defaultValue
	}

	return t.Value
}

func (t *TypeHTTPPath) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeHTTPPath) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeHTTPPath) String() string {
	return t.Value
}
