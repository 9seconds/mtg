package newobfuscated2

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"time"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/newantireplay"
	"github.com/9seconds/mtg/newconfig"
	"github.com/9seconds/mtg/newprotocol"
	"github.com/9seconds/mtg/newwrappers"
)

const clientProtocolHandshakeTimeout = 10 * time.Second

type ClientProtocol struct {
	newprotocol.BaseProtocol
}

func (c *ClientProtocol) Handshake(socket newwrappers.StreamReadWriteCloser) (newwrappers.StreamReadWriteCloser, error) {
	fm, err := c.ReadFrame(socket)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot make client handshake")
	}

	decHasher := sha256.New()
	decHasher.Write(fm.key())           // nolint: errcheck
	decHasher.Write(newconfig.C.Secret) // nolint: errcheck
	decryptor := makeStreamCipher(decHasher.Sum(nil), fm.iv())

	invertedFrame := fm.invert()
	encHasher := sha256.New()
	encHasher.Write(invertedFrame.key()) // nolint: errcheck
	encHasher.Write(newconfig.C.Secret)  // nolint: errcheck
	encryptor := makeStreamCipher(encHasher.Sum(nil), invertedFrame.iv())

	decryptedFrame := frame{}
	decryptor.XORKeyStream(decryptedFrame.bytes(), fm.bytes())

	magic := decryptedFrame.magic()
	switch {
	case bytes.Equal(magic, newprotocol.ConnectionTagAbridged):
		c.ConnectionType = newprotocol.ConnectionTypeAbridged
	case bytes.Equal(magic, newprotocol.ConnectionTagIntermediate):
		c.ConnectionType = newprotocol.ConnectionTypeIntermediate
	case bytes.Equal(magic, newprotocol.ConnectionTagSecure):
		c.ConnectionType = newprotocol.ConnectionTypeSecure
	default:
		return nil, errors.New("Unknown connection type")
	}

	c.ConnectionProtocol = newprotocol.ConnectionProtocolIPv4
	if socket.LocalAddr().IP.To4() == nil {
		c.ConnectionProtocol = newprotocol.ConnectionProtocolIPv6
	}

	buf := bytes.NewReader(decryptedFrame.dc())
	if err := binary.Read(buf, binary.LittleEndian, &c.DC); err != nil {
		c.DC = 1
	}

	antiReplayKey := decryptedFrame.unique()
	if newantireplay.Has(antiReplayKey) {
		return nil, errors.New("Replay attack is detected")
	}
	newantireplay.Add(antiReplayKey)

	return newwrappers.NewObfuscated2(socket, encryptor, decryptor), nil
}

func (c *ClientProtocol) ReadFrame(socket newwrappers.StreamReader) (fm frame, err error) {
	if _, err := io.ReadFull(handshakeReader{socket}, fm.bytes()); err != nil {
		err = errors.Annotate(err, "Cannot extract obfuscated2 frame")
	}
	return
}

type handshakeReader struct {
	parent newwrappers.StreamReader
}

func (h handshakeReader) Read(p []byte) (int, error) {
	return h.parent.ReadTimeout(p, clientProtocolHandshakeTimeout)
}

func makeStreamCipher(key, iv []byte) cipher.Stream {
	block, _ := aes.NewCipher(key) // nolint: gosec
	return cipher.NewCTR(block, iv)
}
