package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	// TypeLogTimeFormatUnix renders log timestamps as integer Unix
	// seconds.
	TypeLogTimeFormatUnix = "unix"

	// TypeLogTimeFormatUnixMs renders log timestamps as integer Unix
	// milliseconds. This is the default and matches mtg's historical
	// behavior.
	TypeLogTimeFormatUnixMs = "unix-ms"

	// TypeLogTimeFormatUnixMicro renders log timestamps as integer Unix
	// microseconds.
	TypeLogTimeFormatUnixMicro = "unix-micro"

	// TypeLogTimeFormatUnixNano renders log timestamps as integer Unix
	// nanoseconds.
	TypeLogTimeFormatUnixNano = "unix-nano"

	// TypeLogTimeFormatRFC3339 renders log timestamps as an RFC3339
	// string, honoring the host timezone.
	TypeLogTimeFormatRFC3339 = "rfc3339"

	// TypeLogTimeFormatRFC3339Nano renders log timestamps as an
	// RFC3339Nano string, honoring the host timezone.
	TypeLogTimeFormatRFC3339Nano = "rfc3339-nano"
)

// TypeLogTimeFormat configures the timestamp format used by the logger.
//
// A value is either one of the presets above or any Go reference-time
// layout (see the time package). Presets map to zerolog's Unix magic
// constants or to time.RFC3339/time.RFC3339Nano; everything else is
// handed to zerolog verbatim as a layout string.
//
// Validation is intentionally shallow: a Go layout is just a string, so
// there is no way to reject a malformed one up front without rendering a
// sample timestamp (and even that has false negatives). Set therefore
// rejects only what is unambiguously wrong — an empty value — and accepts
// any non-empty string, leaving genuine layout typos to surface in the
// log output.
type TypeLogTimeFormat struct {
	Value string
}

func (t *TypeLogTimeFormat) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("log time format cannot be empty")
	}

	t.Value = value

	return nil
}

func (t TypeLogTimeFormat) Get(defaultValue string) string {
	if t.Value == "" {
		return defaultValue
	}

	return t.Value
}

// ZerologFormat resolves the configured value to the string that
// zerolog.TimeFieldFormat expects: a Unix magic constant, an RFC3339
// layout, or the raw Go layout for a free-form value.
func (t TypeLogTimeFormat) ZerologFormat() string {
	switch strings.ToLower(strings.TrimSpace(t.Value)) {
	case TypeLogTimeFormatUnix:
		return zerolog.TimeFormatUnix
	case TypeLogTimeFormatUnixMs:
		return zerolog.TimeFormatUnixMs
	case TypeLogTimeFormatUnixMicro:
		return zerolog.TimeFormatUnixMicro
	case TypeLogTimeFormatUnixNano:
		return zerolog.TimeFormatUnixNano
	case TypeLogTimeFormatRFC3339:
		return time.RFC3339
	case TypeLogTimeFormatRFC3339Nano:
		return time.RFC3339Nano
	default:
		return t.Value
	}
}

func (t *TypeLogTimeFormat) UnmarshalText(data []byte) error {
	return t.Set(string(data))
}

func (t TypeLogTimeFormat) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t TypeLogTimeFormat) String() string {
	return t.Value
}
