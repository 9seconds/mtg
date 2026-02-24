package obfuscation

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type HandshakeFrameTestSuite struct {
	suite.Suite

	frame    handshakeFrame
	reverted handshakeFrame
}

func (h *HandshakeFrameTestSuite) SetupSuite() {
	for i := range hfLen {
		h.frame.data[i] = byte(i + 1)
		h.reverted.data[i] = byte(hfLen - i)
	}
}

func (h *HandshakeFrameTestSuite) TestKey() {
	key := h.frame.key()
	h.EqualValues(8+1, key[0])
	h.EqualValues(8+hfLenKey, key[len(key)-1])
	h.Len(key, hfLenKey)
}

func (h *HandshakeFrameTestSuite) TestIV() {
	iv := h.frame.iv()
	h.EqualValues(40+1, iv[0])
	h.EqualValues(40+hfLenIV, iv[len(iv)-1])
	h.Len(iv, hfLenIV)
}

func (h *HandshakeFrameTestSuite) TestConnectionType() {
	connectionType := h.frame.connectionType()
	h.EqualValues(56+1, connectionType[0])
	h.EqualValues(56+hfLenConnectionType, connectionType[len(connectionType)-1])
	h.Len(connectionType, hfLenConnectionType)
}

func (h *HandshakeFrameTestSuite) TestDCSlice() {
	dcSlice := h.frame.dcSlice()
	h.EqualValues(61, dcSlice[0])
	h.EqualValues(61+1, dcSlice[1])
	h.Len(dcSlice, 2)
}

func (h *HandshakeFrameTestSuite) TestDC() {
	h.Equal(15933, h.frame.dc())
}

func (h *HandshakeFrameTestSuite) TestRevert() {
	fr := h.frame
	fr.revert()

	h.Equal(h.reverted.key(), fr.key())
	h.Equal(h.reverted.iv(), fr.iv())
}

func TestHandshakeFrame(t *testing.T) {
	t.Parallel()
	suite.Run(t, &HandshakeFrameTestSuite{})
}
