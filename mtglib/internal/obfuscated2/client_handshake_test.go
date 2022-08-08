package obfuscated2_test

import (
	"bytes"
	"testing"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/mtglib/internal/obfuscated2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ClientHandshakeTestSuite struct {
	suite.Suite
	SnapshotTestSuite
}

func (suite *ClientHandshakeTestSuite) SetupSuite() {
	suite.NoError(suite.IngestSnapshots(".", "client-handshake-snapshot-"))
}

func (suite *ClientHandshakeTestSuite) TestCannotRead() {
	buf := bytes.NewBuffer([]byte{1, 2, 3})
	_, _, _, err := obfuscated2.ClientHandshake([]byte{1, 2, 3}, buf) //nolint: dogsled

	suite.Error(err)
}

func (suite *ClientHandshakeTestSuite) TestOk() {
	for nameV, snapshotV := range suite.snapshots {
		snapshot := snapshotV

		suite.T().Run(nameV, func(t *testing.T) {
			buf := bytes.NewBuffer(snapshot.Frame.data)

			dc, encryptor, decryptor, err := obfuscated2.ClientHandshake(
				snapshot.Secret.data, buf)
			assert.NoError(t, err)
			assert.EqualValues(t, snapshot.DC, dc)

			writeData := make([]byte, len(snapshot.Encrypted.Text.data))
			readData := make([]byte, len(snapshot.Decrypted.Text.data))

			connMock := &testlib.EssentialsConnMock{}
			connMock.On("Read", mock.Anything).
				Once().
				Return(len(snapshot.Decrypted.Text.data), nil).
				Run(func(args mock.Arguments) {
					arr, ok := args.Get(0).([]byte)

					suite.True(ok)
					copy(arr, snapshot.Decrypted.Cipher.data)
				})
			connMock.On("Write", mock.Anything).
				Once().
				Return(len(snapshot.Encrypted.Text.data), nil).
				Run(func(args mock.Arguments) {
					arr, ok := args.Get(0).([]byte)

					suite.True(ok)
					copy(writeData, arr)
				})

			conn := obfuscated2.Conn{
				Conn:      connMock,
				Encryptor: encryptor,
				Decryptor: decryptor,
			}

			n, err := conn.Read(readData)
			assert.Equal(t, len(readData), n)
			assert.NoError(t, err)
			assert.Equal(t, snapshot.Decrypted.Text.data, readData)

			n, err = conn.Write(snapshot.Encrypted.Text.data)
			assert.Equal(t, len(writeData), n)
			assert.NoError(t, err)
			assert.Equal(t, snapshot.Encrypted.Cipher.data, writeData)

			connMock.AssertExpectations(t)
		})
	}
}

func TestClientHandshake(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ClientHandshakeTestSuite{})
}
