package cli

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestLogTimeFormat(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{value: "", want: zerolog.TimeFormatUnixMs},
		{value: "unix-ms", want: zerolog.TimeFormatUnixMs},
		{value: "unix", want: zerolog.TimeFormatUnix},
		{value: "unix-micro", want: zerolog.TimeFormatUnixMicro},
		{value: "unix-nano", want: zerolog.TimeFormatUnixNano},
		{value: "rfc3339", want: time.RFC3339},
		{value: "rfc3339-nano", want: time.RFC3339Nano},
		{value: "2006-01-02 15:04:05", want: "2006-01-02 15:04:05"},
	}

	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			require.Equal(t, c.want, logTimeFormat(c.value))
		})
	}
}
