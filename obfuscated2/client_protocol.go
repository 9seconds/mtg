package obfuscated2

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/9seconds/mtg/antireplay"
	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/protocol"
	"github.com/9seconds/mtg/utils"
	"github.com/9seconds/mtg/wrappers"
)

const clientProtocolHandshakeTimeout = 10 * time.Second

type ClientProtocol struct {
	protocol.BaseProtocol
}

func (c *ClientProtocol) Handshake(socket wrappers.StreamReadWriteCloser) (wrappers.StreamReadWriteCloser, error) {
	fm, err := c.ReadFrame(socket)
	if err != nil {
		return nil, fmt.Errorf("cannot make a client handshake: %w", err)
	}

	decHasher := sha256.New()
	decHasher.Write(fm.Key())        // nolint: errcheck
	decHasher.Write(config.C.Secret) // nolint: errcheck
	decryptor := utils.MakeStreamCipher(decHasher.Sum(nil), fm.IV())

	invertedFrame := fm.Invert()
	encHasher := sha256.New()
	encHasher.Write(invertedFrame.Key()) // nolint: errcheck
	encHasher.Write(config.C.Secret)     // nolint: errcheck
	encryptor := utils.MakeStreamCipher(encHasher.Sum(nil), invertedFrame.IV())

	decryptedFrame := Frame{}
	decryptor.XORKeyStream(decryptedFrame.Bytes(), fm.Bytes())

	magic := decryptedFrame.Magic()
	switch {
	case bytes.Equal(magic, conntypes.ConnectionTagAbridged):
		c.ConnectionType = conntypes.ConnectionTypeAbridged
	case bytes.Equal(magic, conntypes.ConnectionTagIntermediate):
		c.ConnectionType = conntypes.ConnectionTypeIntermediate
	case bytes.Equal(magic, conntypes.ConnectionTagSecure):
		c.ConnectionType = conntypes.ConnectionTypeSecure
	default:
		return nil, errors.New("Unknown connection type")
	}

	c.ConnectionProtocol = conntypes.ConnectionProtocolIPv4
	if socket.LocalAddr().IP.To4() == nil {
		c.ConnectionProtocol = conntypes.ConnectionProtocolIPv6
	}

	buf := bytes.NewReader(decryptedFrame.DC())
	if err := binary.Read(buf, binary.LittleEndian, &c.DC); err != nil {
		c.DC = conntypes.DCDefaultIdx
	}

	antiReplayKey := decryptedFrame.Unique()
	if antireplay.Has(antiReplayKey) {
		return nil, errors.New("Replay attack is detected")
	}
	antireplay.Add(antiReplayKey)

	return wrappers.NewObfuscated2(socket, encryptor, decryptor), nil
}

func (c *ClientProtocol) ReadFrame(socket wrappers.StreamReader) (fm Frame, err error) {
	if _, err = io.ReadFull(handshakeReader{socket}, fm.Bytes()); err != nil {
		err = fmt.Errorf("cannot extract obfuscated2 frame: %w", err)
	}
	return
}

type handshakeReader struct {
	parent wrappers.StreamReader
}

func (h handshakeReader) Read(p []byte) (int, error) {
	return h.parent.ReadTimeout(p, clientProtocolHandshakeTimeout)
}

func MakeClientProtocol() protocol.ClientProtocol {
	return &ClientProtocol{}
}
