package config

import (
	"fmt"
	"strings"
)

const (
	TypeTLSProfileChrome  = "chrome"
	TypeTLSProfileFirefox = "firefox"
	TypeTLSProfileSafari  = "safari"
	TypeTLSProfileEdge    = "edge"
)

type TypeTLSProfile struct {
	Value string
}

func (t *TypeTLSProfile) Set(value string) error {
	value = strings.ToLower(value)

	switch value {
	case TypeTLSProfileChrome, TypeTLSProfileFirefox,
		TypeTLSProfileSafari, TypeTLSProfileEdge:
		t.Value = value

		return nil
	case "":
		return nil
	default:
		return fmt.Errorf("unsupported tls profile: %s (supported: chrome, firefox, safari, edge)", value)
	}
}

func (t *TypeTLSProfile) Get(defaultValue string) string {
	if t.Value == "" {
		return defaultValue
	}

	return t.Value
}

func (t *TypeTLSProfile) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeTLSProfile) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeTLSProfile) String() string {
	return t.Value
}
