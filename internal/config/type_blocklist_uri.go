package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

type TypeBlocklistURI struct {
	Value string
}

func (t *TypeBlocklistURI) Set(value string) error {
	if stat, err := os.Stat(value); err == nil || os.IsExist(err) {
		switch {
		case stat.IsDir():
			return fmt.Errorf("value is correct filepath but directory")
		case stat.Mode().Perm()&0o400 == 0:
			return fmt.Errorf("value is correct filepath but not readable")
		}

		value, err = filepath.Abs(value)
		if err != nil {
			return fmt.Errorf(
				"value is correct filepath but cannot resolve absolute (%s): %w",
				value, err)
		}

		t.Value = value

		return nil
	}

	parsedURL, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("incorrect url (%s): %w", value, err)
	}

	switch parsedURL.Scheme {
	case "http", "https":
	default:
		return fmt.Errorf("unknown schema %s (%s)", parsedURL.Scheme, value)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("incorrect url %s", value)
	}

	t.Value = parsedURL.String()

	return nil
}

func (t TypeBlocklistURI) Get(defaultValue string) string {
	if t.Value == "" {
		return defaultValue
	}

	return t.Value
}

func (t TypeBlocklistURI) IsRemote() bool {
	return !filepath.IsAbs(t.Value)
}

func (t *TypeBlocklistURI) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeBlocklistURI) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeBlocklistURI) String() string {
	return t.Value
}
