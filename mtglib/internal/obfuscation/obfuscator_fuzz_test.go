package obfuscation_test

import (
	"bytes"
	"testing"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/mtglib/internal/obfuscation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func FuzzClientServerHandshakes(f *testing.F) {
	f.Add(int16(1), make([]byte, mtglib.SecretKeyLength))

	f.Fuzz(func(t *testing.T, dc int16, data []byte) {
		if dc <= 0 {
			dc = 1
		}

		client := obfuscation.Obfuscator{
			Secret: data,
		}
		server := client

		clientToServerBuf := &bytes.Buffer{}

		writeConnMock := &testlib.EssentialsConnMock{}
		writeConnMock.
			On("Write", mock.AnythingOfType("[]uint8")).
			Once().
			Return(64, nil).
			Run(func(args mock.Arguments) {
				arg := args.Get(0).([]byte)
				n, err := clientToServerBuf.Write(arg)
				assert.Equal(t, 64, n)
				assert.NoError(t, err)
			})

		readConnMock := &testlib.EssentialsConnMock{}
		readConnMock.
			On("Read", mock.AnythingOfType("[]uint8")).
			Once().
			Return(64, nil).
			Run(func(args mock.Arguments) {
				arg := args.Get(0).([]byte)
				n, err := clientToServerBuf.Read(arg)
				assert.Equal(t, 64, n)
				assert.NoError(t, err)
			})

		_, err := client.SendHandshake(writeConnMock, int(dc))
		assert.NoError(t, err)

		readDc, _, err := server.ReadHandshake(readConnMock)
		assert.NoError(t, err)
		assert.EqualValues(t, dc, readDc)

		writeConnMock.AssertExpectations(t)
		readConnMock.AssertExpectations(t)
	})
}
