package obfuscated2_test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ServerHandshakeTestSuite struct {
	suite.Suite

	data ServerHandshakeTestData
}

func (suite *ServerHandshakeTestSuite) SetupTest() {
	suite.data = NewServerHandshakeTestData(suite.T())
}

func (suite *ServerHandshakeTestSuite) TearDownTest() {
	suite.data.connMock.AssertExpectations(suite.T())
}

func (suite *ServerHandshakeTestSuite) TestSendToTelegram() {
	messageToTelegram := []byte{10, 11, 12, 13, 14, 'a'}

	suite.data.connMock.
		On("Write", mock.Anything).
		Return(len(messageToTelegram), nil).
		Once().
		Run(func(args mock.Arguments) {
			message := make([]byte, len(messageToTelegram))
			suite.data.decryptor.XORKeyStream(message, args.Get(0).([]byte)) //nolint: forcetypeassert
			suite.Equal(messageToTelegram, message)
		})

	n, err := suite.data.proxyConn.Write(messageToTelegram)
	suite.EqualValues(len(messageToTelegram), n)
	suite.NoError(err)
}

func (suite *ServerHandshakeTestSuite) TestRecieveFromTelegram() {
	messageFromTelegram := []byte{10, 11, 12, 13, 14, 'a'}
	buffer := make([]byte, len(messageFromTelegram))

	suite.data.connMock.
		On("Read", mock.Anything).
		Return(len(messageFromTelegram), nil).
		Once().
		Run(func(args mock.Arguments) {
			message := make([]byte, len(messageFromTelegram))
			suite.data.encryptor.XORKeyStream(message, messageFromTelegram)
			copy(args.Get(0).([]byte), message) //nolint: forcetypeassert
		})

	n, err := suite.data.proxyConn.Read(buffer)
	suite.EqualValues(len(messageFromTelegram), n)
	suite.NoError(err)
	suite.Equal(messageFromTelegram, buffer)
}

func TestServerHandshake(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ServerHandshakeTestSuite{})
}
