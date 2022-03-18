package faketls_test

import (
	"testing"

	"github.com/9seconds/mtg/v2/mtglib/internal/faketls"
	"github.com/stretchr/testify/require"
)

var FuzzClientHelloSecret = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

func FuzzClientHello(f *testing.F) {
	f.Add([]byte{1, 2, 3})

	f.Fuzz(func(t *testing.T, frame []byte) {
		_, err := faketls.ParseClientHello(FuzzClientHelloSecret, frame)

		// a probability of having != err is almost negligible
		require.Error(t, err)
	})
}
