package mtglib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitLength(t *testing.T) {
	// A TLS record header announcing 122 bytes of payload, then more data.
	response := make([]byte, 3000)
	copy(response, []byte{0x16, 0x03, 0x03, 0x00, 0x7a})

	cases := []struct {
		name   string
		data   []byte
		prefix int
		want   int
	}{
		{name: "default is the first record", data: response, prefix: 0, want: 5 + 122},
		{name: "negative is the default", data: response, prefix: -1, want: 5 + 122},
		{name: "explicit prefix", data: response, prefix: 300, want: 300},
		{name: "prefix above the response", data: response, prefix: 1 << 16, want: len(response)},
		{name: "record longer than data", data: response[:100], prefix: 0, want: 100},
		{name: "no complete header", data: response[:4], prefix: 0, want: 4},
		{name: "empty", data: nil, prefix: 0, want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, splitLength(tc.data, tc.prefix))
		})
	}
}
