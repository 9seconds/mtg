package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

type TypeBlocklistURI struct {
	value string
}

func (c *TypeBlocklistURI) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	text := string(data)
	if filepath.IsAbs(text) {
		if _, err := os.Stat(text); os.IsNotExist(err) {
			return fmt.Errorf("filepath %s does not exist", text)
		}

		c.value = text

		return nil
	}

	parsedURL, err := url.Parse(text)
	if err != nil {
		return fmt.Errorf("incorrect url: %w", err)
	}

	switch parsedURL.Scheme {
	case "http", "https": // nolint: goconst
	default:
		return fmt.Errorf("unknown schema %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("incorrect url %s", text)
	}

	c.value = parsedURL.String()

	return nil
}

func (c TypeBlocklistURI) MarshalText() ([]byte, error) {
	return []byte(c.value), nil
}

func (c TypeBlocklistURI) String() string {
	return c.value
}

func (c TypeBlocklistURI) IsRemote() bool {
	return !filepath.IsAbs(c.value)
}

func (c TypeBlocklistURI) Value(defaultValue string) string {
	if c.value == "" {
		return defaultValue
	}

	return c.value
}
