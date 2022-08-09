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
	"github.com/9seconds/mtg/stats"
	"github.com/9seconds/mtg/utils"
	"github.com/9seconds/mtg/wrappers/stream"
)

const clientProtocolHandshakeTimeout = 10 * time.Second

type ClientProtocol struct {
	connectionType     conntypes.ConnectionType
	connectionProtocol conntypes.ConnectionProtocol
	dc                 conntypes.DC
}

func (c *ClientProtocol) ConnectionType() conntypes.ConnectionType {
	return c.connectionType
}

func (c *ClientProtocol) ConnectionProtocol() conntypes.ConnectionProtocol {
	return c.connectionProtocol
}

func (c *ClientProtocol) DC() conntypes.DC {
	return c.dc
}

func (c *ClientProtocol) Handshake(socket conntypes.StreamReadWriteCloser) (conntypes.StreamReadWriteCloser, error) {
	fm, err := c.ReadFrame(socket)
	if err != nil {
		return nil, fmt.Errorf("cannot make a client handshake: %w", err)
	}

	decHasher := sha256.New()
	decHasher.Write(fm.Key())
	decHasher.Write(config.C.Secret)
	decryptor := utils.MakeStreamCipher(decHasher.Sum(nil), fm.IV())

	invertedFrame := fm.Invert()
	encHasher := sha256.New()
	encHasher.Write(invertedFrame.Key())
	encHasher.Write(config.C.Secret)
	encryptor := utils.MakeStreamCipher(encHasher.Sum(nil), invertedFrame.IV())

	decryptedFrame := Frame{}
	decryptor.XORKeyStream(decryptedFrame.Bytes(), fm.Bytes())

	magic := decryptedFrame.Magic()

	switch {
	case bytes.Equal(magic, conntypes.ConnectionTagAbridged):
		c.connectionType = conntypes.ConnectionTypeAbridged
	case bytes.Equal(magic, conntypes.ConnectionTagIntermediate):
		c.connectionType = conntypes.ConnectionTypeIntermediate
	case bytes.Equal(magic, conntypes.ConnectionTagSecure):
		c.connectionType = conntypes.ConnectionTypeSecure
	default:
		return nil, errors.New("unknown connection type")
	}

	c.connectionProtocol = conntypes.ConnectionProtocolIPv4
	if socket.LocalAddr().IP.To4() == nil {
		c.connectionProtocol = conntypes.ConnectionProtocolIPv6
	}

	buf := bytes.NewReader(decryptedFrame.DC())
	if err := binary.Read(buf, binary.LittleEndian, &c.dc); err != nil {
		c.dc = conntypes.DCDefaultIdx
	}

	replayKey := decryptedFrame.Unique()
	if antireplay.Cache.HasObfuscated2(replayKey) {
		stats.Stats.ReplayDetected()

		return nil, errors.New("replay attack is detected")
	}

	antireplay.Cache.AddObfuscated2(replayKey)

	return stream.NewObfuscated2(socket, encryptor, decryptor), nil
}

func (c *ClientProtocol) ReadFrame(socket conntypes.StreamReader) (Frame, error) {
	fm := Frame{}

	if _, err := io.ReadFull(handshakeReader{socket}, fm.Bytes()); err != nil {
		return fm, fmt.Errorf("cannot extract obfuscated2 frame: %w", err)
	}

	return fm, nil
}

type handshakeReader struct {
	parent conntypes.StreamReader
}

func (h handshakeReader) Read(p []byte) (int, error) {
	return h.parent.ReadTimeout(p, clientProtocolHandshakeTimeout) //nolint: wrapcheck
}

func MakeClientProtocol() protocol.ClientProtocol {
	return &ClientProtocol{}
}
