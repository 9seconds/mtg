package obfuscated2_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"testing"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/mtglib/internal/obfuscated2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ServerHandshakeTestSuite struct {
	suite.Suite

	connMock  *testlib.NetConnMock
	proxyConn obfuscated2.Conn
	encryptor cipher.Stream
	decryptor cipher.Stream
}

func (suite *ServerHandshakeTestSuite) SetupTest() {
	buf := &bytes.Buffer{}
	suite.connMock = &testlib.NetConnMock{}

	encryptor, decryptor, err := obfuscated2.ServerHandshake(buf)
	suite.NoError(err)

	suite.proxyConn = obfuscated2.Conn{
		Conn:      suite.connMock,
		Encryptor: encryptor,
		Decryptor: decryptor,
	}

	serverEncrypted := buf.Bytes()

	decBlock, _ := aes.NewCipher(serverEncrypted[8 : 8+32])
	suite.decryptor = cipher.NewCTR(decBlock, serverEncrypted[8+32:8+32+16])

	serverDecrypted := make([]byte, len(serverEncrypted))
	suite.decryptor.XORKeyStream(serverDecrypted, serverEncrypted)

	suite.Equal("3d3d3Q",
		base64.RawStdEncoding.EncodeToString(serverDecrypted[8+32+16:8+32+16+4]))

	serverEncryptedReverted := make([]byte, len(serverEncrypted))

	for i := 0; i < 32+16; i++ {
		serverEncryptedReverted[8+i] = serverEncrypted[8+32+16-1-i]
	}

	encBlock, _ := aes.NewCipher(serverEncryptedReverted[8 : 8+32])
	suite.encryptor = cipher.NewCTR(encBlock, serverEncryptedReverted[8+32:8+32+16])
}

func (suite *ServerHandshakeTestSuite) TearDownTest() {
	suite.connMock.AssertExpectations(suite.T())
}

func (suite *ServerHandshakeTestSuite) TestSendToTelegram() {
	messageToTelegram := []byte{10, 11, 12, 13, 14, 'a'}

	suite.connMock.
		On("Write", mock.Anything).
		Return(len(messageToTelegram), nil).
		Once().
		Run(func(args mock.Arguments) {
			message := make([]byte, len(messageToTelegram))
			suite.decryptor.XORKeyStream(message, args.Get(0).([]byte))
			suite.Equal(messageToTelegram, message)
		})

	n, err := suite.proxyConn.Write(messageToTelegram)
	suite.EqualValues(len(messageToTelegram), n)
	suite.NoError(err)
}

func (suite *ServerHandshakeTestSuite) TestRecieveFromTelegram() {
	messageFromTelegram := []byte{10, 11, 12, 13, 14, 'a'}
	buffer := make([]byte, len(messageFromTelegram))

	suite.connMock.
		On("Read", mock.Anything).
		Return(len(messageFromTelegram), nil).
		Once().
		Run(func(args mock.Arguments) {
			message := make([]byte, len(messageFromTelegram))
			suite.encryptor.XORKeyStream(message, messageFromTelegram)
			copy(args.Get(0).([]byte), message)
		})

	n, err := suite.proxyConn.Read(buffer)
	suite.EqualValues(len(messageFromTelegram), n)
	suite.NoError(err)
	suite.Equal(messageFromTelegram, buffer)
}

func TestServerHandshake(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ServerHandshakeTestSuite{})
}
