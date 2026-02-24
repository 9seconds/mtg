package obfuscation

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	"github.com/9seconds/mtg/v2/essentials"
)

type Obfuscator struct {
	Secret []byte
}

func (o Obfuscator) ReadHandshake(r essentials.Conn) (int, essentials.Conn, error) {
	frame := handshakeFrame{}

	if _, err := io.ReadFull(r, frame.data[:]); err != nil {
		return 0, nil, fmt.Errorf("cannot read frame: %w", err)
	}

	hasher := sha256.New()
	recvCipher := o.getCipher(&frame, hasher)

	frame.revert()
	hasher.Reset()
	sendCipher := o.getCipher(&frame, hasher)

	recvCipher.XORKeyStream(frame.data[:], frame.data[:])

	if val := frame.connectionType(); subtle.ConstantTimeCompare(val, hfConnectionType[:]) != 1 {
		return 0, nil, fmt.Errorf("unsupported connection type: %s", hex.EncodeToString(val))
	}

	cn := conn{
		Conn:       r,
		recvCipher: recvCipher,
		sendCipher: sendCipher,
	}

	return frame.dc(), cn, nil
}

func (o Obfuscator) SendHandshake(w essentials.Conn, dc int) (essentials.Conn, error) {
	frame := generateHandshake(dc)
	copyFrame := frame
	hasher := sha256.New()

	sendCipher := o.getCipher(&frame, hasher)

	frame.revert()
	hasher.Reset()
	recvCipher := o.getCipher(&frame, hasher)

	sendCipher.XORKeyStream(frame.data[:], frame.data[:])
	copy(frame.key(), copyFrame.key())
	copy(frame.iv(), copyFrame.iv())

	if _, err := w.Write(frame.data[:]); err != nil {
		return nil, fmt.Errorf("cannot send a handshake: %w", err)
	}

	return conn{
		Conn:       w,
		recvCipher: recvCipher,
		sendCipher: sendCipher,
	}, nil
}

func (o Obfuscator) getCipher(f *handshakeFrame, hasher hash.Hash) cipher.Stream {
	blockKey := f.key()

	if o.Secret != nil {
		hasher.Write(blockKey)
		hasher.Write(o.Secret)
		blockKey = hasher.Sum(nil)
	}

	block, _ := aes.NewCipher(blockKey)

	return cipher.NewCTR(block, f.iv())
}
