package fake_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/dolonet/mtg-multi/internal/testlib"
	"github.com/dolonet/mtg-multi/mtglib"
	"github.com/dolonet/mtg-multi/mtglib/internal/tls/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type connMock struct {
	testlib.EssentialsConnMock

	readBuf *bytes.Buffer
}

func (f *connMock) Read(p []byte) (int, error) {
	return f.readBuf.Read(p)
}

func FuzzReadClientHello(f *testing.F) {
	seed := [248]byte{}

	secret, err := mtglib.ParseSecret(
		"ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d",
	)
	require.NoError(f, err)

	f.Add(seed[:])

	f.Fuzz(func(t *testing.T, value []byte) {
		r := &connMock{
			readBuf: bytes.NewBuffer(value),
		}
		r.
			On("SetReadDeadline", mock.AnythingOfType("time.Time")).
			Twice().
			Return(nil)

		_, err := fake.ReadClientHello(r, secret.Key[:], secret.Host, time.Hour)
		assert.Error(t, err)
	})
}
