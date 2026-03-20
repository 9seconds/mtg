package config

import (
	"fmt"
	"net/url"
)

type TypeURL struct {
	Value *url.URL
}

func (t *TypeURL) Set(value string) error {
	parsedURL, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("value is not correct URL (%s): %w", value, err)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("url has to have a schema: %s", value)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported schema: %s", parsedURL.Scheme)
	}

	t.Value = parsedURL

	return nil
}

func (t *TypeURL) Get(defaultValue *url.URL) *url.URL {
	if t.Value == nil {
		return defaultValue
	}

	return t.Value
}

func (t *TypeURL) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeURL) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeURL) String() string {
	if t.Value == nil {
		return ""
	}

	return t.Value.String()
}
