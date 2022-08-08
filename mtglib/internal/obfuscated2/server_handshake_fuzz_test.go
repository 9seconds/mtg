package obfuscated2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func FuzzServerSend(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5})

	f.Fuzz(func(t *testing.T, data []byte) {
		handshakeData := NewServerHandshakeTestData(t)

		handshakeData.connMock.
			On("Write", mock.Anything).
			Return(len(data), nil).
			Once().
			Run(func(args mock.Arguments) {
				message := make([]byte, len(data))
				handshakeData.decryptor.XORKeyStream(message, args.Get(0).([]byte)) //nolint: forcetypeassert
				assert.Equal(t, message, data)
			})

		n, err := handshakeData.proxyConn.Write(data)

		assert.EqualValues(t, len(data), n)
		assert.NoError(t, err)
		handshakeData.connMock.AssertExpectations(t)
	})
}

func FuzzServerReceive(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5})

	f.Fuzz(func(t *testing.T, data []byte) {
		handshakeData := NewServerHandshakeTestData(t)
		buffer := make([]byte, len(data))

		handshakeData.connMock.
			On("Read", mock.Anything).
			Return(len(data), nil).
			Once().
			Run(func(args mock.Arguments) {
				message := make([]byte, len(data))
				handshakeData.encryptor.XORKeyStream(message, data)
				copy(args.Get(0).([]byte), message) //nolint: forcetypeassert
			})

		n, err := handshakeData.proxyConn.Read(buffer)

		assert.EqualValues(t, len(data), n)
		assert.NoError(t, err)
		assert.Equal(t, data, buffer)
		handshakeData.connMock.AssertExpectations(t)
	})
}
