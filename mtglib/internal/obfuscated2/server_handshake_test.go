package obfuscated2_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"testing"

	"github.com/9seconds/mtg/v2/mtglib/internal/obfuscated2"
	"github.com/9seconds/mtg/v2/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ServerHandshakeTestSuite struct {
	suite.Suite
}

func (suite *ServerHandshakeTestSuite) TestOk() {
	buf := &bytes.Buffer{}
	connMock := &testlib.NetConnMock{}

	encryptor, decryptor, err := obfuscated2.ServerHandshake(buf)
	suite.NotNil(encryptor)
	suite.NotNil(decryptor)
	suite.NoError(err)

	proxyConn := &obfuscated2.Conn{
		Conn:      connMock,
		Encryptor: encryptor,
		Decryptor: decryptor,
	}

	serverEncrypted := buf.Bytes()

	decBlock, _ := aes.NewCipher(serverEncrypted[8 : 8+32])
	serverDecryptor := cipher.NewCTR(decBlock, serverEncrypted[8+32:8+32+16])
	serverDecrypted := make([]byte, len(serverEncrypted))
	serverDecryptor.XORKeyStream(serverDecrypted, serverEncrypted)

	suite.Equal("3d3d3Q",
		base64.RawStdEncoding.EncodeToString(serverDecrypted[8+32+16:8+32+16+4]))

	serverEncryptedReverted := make([]byte, len(serverEncrypted))

	for i := 0; i < 32+16; i++ {
		serverEncryptedReverted[8+i] = serverEncrypted[8+32+16-1-i]
	}

	encBlock, _ := aes.NewCipher(serverEncryptedReverted[8 : 8+32])
	serverEncryptor := cipher.NewCTR(encBlock, serverEncryptedReverted[8+32:8+32+16])

	messageFromTelegram := []byte{1, 2, 3, 4, 5}
	// messageToTelegram := []byte{10, 11, 13, 14}
	bufferToRead := make([]byte, 5)

	connMock.
		On("Read", mock.Anything).
		Return(5, nil).
		Once().
		Run(func(args mock.Arguments) {
			messageToRead := make([]byte, len(messageFromTelegram))
			serverEncryptor.XORKeyStream(messageToRead, messageFromTelegram)
			copy(args.Get(0).([]byte), messageToRead)
		})

	n, err := proxyConn.Read(bufferToRead)
	suite.EqualValues(5, n)
	suite.NoError(err)
	suite.Equal(messageFromTelegram, bufferToRead)

	messageToTelegram := []byte{10, 11, 12, 13, 14}

	connMock.
		On("Write", mock.Anything).
		Return(5, nil).
		Once().
		Run(func(args mock.Arguments) {
			message := make([]byte, len(messageToTelegram))
			serverDecryptor.XORKeyStream(message, args.Get(0).([]byte))
			suite.Equal(messageToTelegram, message)
		})

	n, err = proxyConn.Write(messageToTelegram)
	suite.EqualValues(5, n)
	suite.NoError(err)

	connMock.AssertExpectations(suite.T())
}

func TestServerHandshake(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ServerHandshakeTestSuite{})
}
