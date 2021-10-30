package config

import (
	"fmt"
	"net/url"
)

type TypeURLHost struct {
	Value string
}

func (t *TypeURLHost) Set(value string) error {

	if value == "" {
		return fmt.Errorf("value empty")
	}

	testUrl := fmt.Sprintf("https://%s/dns-query", value)
	p, err := url.Parse(testUrl)
	if err != nil {
		return err
	}

	if p.Host != value {
		return fmt.Errorf("value is not a valid url host: %s", value)
	}

	t.Value = value

	return nil
}

func (t TypeURLHost) Get(defaultValue string) string {
	if t.Value == "" {
		return defaultValue
	}

	return t.Value
}

func (t *TypeURLHost) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeURLHost) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeURLHost) String() string {
	return t.Value
}
