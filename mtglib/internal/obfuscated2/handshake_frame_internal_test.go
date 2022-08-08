package obfuscated2

import (
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type HandshakeFrameTestSuite struct {
	suite.Suite
}

func (suite *HandshakeFrameTestSuite) Decode(value string) []byte {
	v, err := base64.RawStdEncoding.DecodeString(value)
	suite.NoError(err)

	return v
}

func (suite *HandshakeFrameTestSuite) Encode(value []byte) string {
	return base64.RawStdEncoding.EncodeToString(value)
}

func (suite *HandshakeFrameTestSuite) TestOk() {
	hf := handshakeFrame{}
	testFrame := suite.Decode(
		"L9TmCzzxl9bPKODBpZeVM/qqNUxQ/axxBup1S2ymbIfUd6f7YSyzzM9EmTFv2/XzGqJGEHuj2zofmUGBLghu5g")
	copy(hf.data[:], testFrame)

	suite.Equal("zyjgwaWXlTP6qjVMUP2scQbqdUtspmyH1Hen+2Ess8w", suite.Encode(hf.key()))
	suite.Equal("z0SZMW/b9fMaokYQe6PbOg", suite.Encode(hf.iv()))
	suite.Equal("H5lBgQ", suite.Encode(hf.connectionType()))
	suite.EqualValues(2094, hf.dc())

	inverted := hf.invert()
	suite.Equal("OtujexBGohrz9dtvMZlEz8yzLGH7p3fUh2ymbEt16gY", suite.Encode(inverted.key()))
	suite.Equal("caz9UEw1qvozlZelweAozw", suite.Encode(inverted.iv()))
	suite.Equal("H5lBgQ", suite.Encode(inverted.connectionType()))
	suite.EqualValues(2094, inverted.dc())
}

func (suite *HandshakeFrameTestSuite) TestDC() {
	testData := map[int16]int{
		1:  1,
		-1: 1,
		0:  DefaultDC,
	}

	for k, v := range testData {
		incoming := k
		expected := v

		suite.T().Run(strconv.Itoa(int(incoming)), func(t *testing.T) {
			frame := handshakeFrame{}

			rand.Read(frame.data[:]) //nolint: errcheck

			frame.data[handshakeFrameOffsetDC] = byte(incoming)
			frame.data[handshakeFrameOffsetDC+1] = byte(incoming >> 8)

			assert.Equal(t, expected, frame.dc())
		})
	}
}

func TestHandshakeFrame(t *testing.T) {
	t.Parallel()
	suite.Run(t, &HandshakeFrameTestSuite{})
}
