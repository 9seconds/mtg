package cli

import (
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestMakeLoggerTimeFieldFormat verifies that the configured logTimeFormat
// is what ends up in zerolog's global TimeFieldFormat — including the
// historical default when the field is unset.
func TestMakeLoggerTimeFieldFormat(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "unset-default", value: "", want: zerolog.TimeFormatUnixMs},
		{name: "unix", value: "unix", want: zerolog.TimeFormatUnix},
		{name: "unix-ms", value: "unix-ms", want: zerolog.TimeFormatUnixMs},
		{name: "unix-micro", value: "unix-micro", want: zerolog.TimeFormatUnixMicro},
		{name: "unix-nano", value: "unix-nano", want: zerolog.TimeFormatUnixNano},
		{name: "rfc3339", value: "rfc3339", want: time.RFC3339},
		{name: "rfc3339-nano", value: "rfc3339-nano", want: time.RFC3339Nano},
		{name: "go-layout", value: "2006-01-02 15:04:05", want: "2006-01-02 15:04:05"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			conf := &config.Config{}
			if c.value != "" {
				assert.NoError(t, conf.LogTimeFormat.Set(c.value))
			}

			makeLogger(conf)
			assert.Equal(t, c.want, zerolog.TimeFieldFormat)
		})
	}
}
