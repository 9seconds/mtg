package config

import (
	"fmt"
	"net/url"
)

type TypeHttpsURL struct {
	Value *url.URL
}

func (t *TypeHttpsURL) Set(value string) error {
	parsedURL, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("value is not correct URL (%s): %w", value, err)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("url has to have a schema: %s", value)
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported schema: %s", parsedURL.Scheme)
	}

	t.Value = parsedURL

	return nil
}

func (t *TypeHttpsURL) Get(defaultValue *url.URL) *url.URL {
	if t.Value == nil {
		return defaultValue
	}

	return t.Value
}

func (t *TypeHttpsURL) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeHttpsURL) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeHttpsURL) String() string {
	if t.Value == nil {
		return ""
	}

	return t.Value.String()
}
