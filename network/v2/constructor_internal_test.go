package network

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/stretchr/testify/require"
)

// New keeps the exact v2.2.8 function type, including assignments by callers.
var _ func(*net.Resolver, string, time.Duration, time.Duration, time.Duration,
	net.KeepAliveConfig) mtglib.Network = New

func TestConstructorCompatibility(t *testing.T) {
	t.Parallel()

	legacy, ok := New(nil, "", 0, 0, 0, DefaultKeepAliveConfig).(*network)
	require.True(t, ok)
	require.Equal(t, DefaultTCPNotSentLowat, legacy.tcpNotSentLowat)

	for _, value := range []int{0, -1, 4096} {
		t.Run(strconv.Itoa(value), func(t *testing.T) {
			current, ok := NewWithTCPNotSentLowat(nil, "", 0, 0, 0,
				DefaultKeepAliveConfig, value).(*network)
			require.True(t, ok)
			want := value
			if value == 0 {
				want = DefaultTCPNotSentLowat
			}
			require.Equal(t, want, current.tcpNotSentLowat)
		})
	}
}
