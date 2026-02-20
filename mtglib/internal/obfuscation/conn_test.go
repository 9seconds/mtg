package obfuscation

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"testing"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ConnTestSuite struct {
	suite.Suite

	secret []byte
}

func (s *ConnTestSuite) SetupSuite() {
	secret := [32]byte{}
	s.secret = secret[:]
}

func (s *ConnTestSuite) TestRead() {
	testData := map[string]string{
		"data1": "b8f4b41993",
		"":      "",
		"___":   "83ca9f",
	}

	for incoming, outgoing := range testData {
		s.T().Run(incoming, func(t *testing.T) {
			connMock := &testlib.EssentialsConnMock{}
			testConn := s.makeConn(connMock)
			data := make([]byte, len(incoming))

			connMock.On("Read", make([]byte, len(incoming))).Return(len(incoming), nil).Run(func(args mock.Arguments) {
				arg := args.Get(0).([]byte)
				copy(arg, []byte(incoming))
			})

			n, err := testConn.Read(data)

			assert.Equal(t, len(data), n)
			assert.NoError(t, err)
			assert.Equal(t, outgoing, hex.EncodeToString(data))

			connMock.AssertExpectations(t)
		})
	}
}

func (s *ConnTestSuite) TestWrite() {
	testData := map[string]string{
		"b8f4b41993": "data1",
		"":           "",
		"83ca9f":     "___",
	}

	for incoming, outgoing := range testData {
		s.T().Run(incoming, func(t *testing.T) {
			connMock := &testlib.EssentialsConnMock{}
			testConn := s.makeConn(connMock)
			toWrite, _ := hex.DecodeString(incoming)
			data := make([]byte, len(toWrite))

			connMock.On("Write", []byte(outgoing)).Return(len(toWrite), nil)

			n, err := testConn.Write(toWrite)
			assert.Equal(t, len(data), n)
			assert.NoError(t, err)

			connMock.AssertExpectations(t)
		})
	}
}

func (s *ConnTestSuite) makeConn(rawConn *testlib.EssentialsConnMock) essentials.Conn {
	rblock, err := aes.NewCipher(s.secret)
	if err != nil {
		panic(err)
	}

	wblock, err := aes.NewCipher(s.secret)
	if err != nil {
		panic(err)
	}

	return conn{
		Conn:       rawConn,
		sendCipher: cipher.NewCTR(wblock, s.secret[:aes.BlockSize]),
		recvCipher: cipher.NewCTR(rblock, s.secret[:aes.BlockSize]),
	}
}

func TestConn(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConnTestSuite{})
}
