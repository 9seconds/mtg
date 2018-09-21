package obfuscated2

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
)

// Obfuscated2 contains AES CTR encryption and decryption streams
// for telegram connection.
type Obfuscated2 struct {
	Decryptor cipher.Stream
	Encryptor cipher.Stream
}

// ParseObfuscated2ClientFrame parses client frame. Please check this link for
// details: http://telegra.ph/telegram-blocks-wtf-05-26
//
// Beware, link above is in russian.
func ParseObfuscated2ClientFrame(secret []byte, frame Frame) (*Obfuscated2, *mtproto.ConnectionOpts, error) {
	decHasher := sha256.New()
	decHasher.Write(frame.Key()) // nolint: errcheck, gosec
	decHasher.Write(secret)      // nolint: errcheck, gosec
	decryptor := makeStreamCipher(decHasher.Sum(nil), frame.IV())

	invertedFrame := frame.Invert()
	encHasher := sha256.New()
	encHasher.Write(invertedFrame.Key()) // nolint: errcheck, gosec
	encHasher.Write(secret)              // nolint: errcheck, gosec
	encryptor := makeStreamCipher(encHasher.Sum(nil), invertedFrame.IV())

	decryptedFrame := make(Frame, FrameLen)
	decryptor.XORKeyStream(decryptedFrame, frame)
	connType, err := decryptedFrame.ConnectionType()
	if err != nil {
		return nil, nil, errors.Annotate(err, "Unknown protocol")
	}

	obfs := &Obfuscated2{
		Decryptor: decryptor,
		Encryptor: encryptor,
	}
	connOpts := &mtproto.ConnectionOpts{
		DC:             decryptedFrame.DC(),
		ConnectionType: connType,
	}

	return obfs, connOpts, nil
}

// MakeTelegramObfuscated2Frame creates new handshake frame to send to
// Telegram.
// https://blog.susanka.eu/how-telegram-obfuscates-its-mtproto-traffic/
func MakeTelegramObfuscated2Frame(opts *mtproto.ConnectionOpts) (*Obfuscated2, Frame) {
	frame := generateFrame(opts.ConnectionType)

	encryptor := makeStreamCipher(frame.Key(), frame.IV())
	decryptorFrame := frame.Invert()
	decryptor := makeStreamCipher(decryptorFrame.Key(), decryptorFrame.IV())

	copyFrame := make(Frame, FrameLen)
	copy(copyFrame[:frameOffsetIV], frame[:frameOffsetIV])
	encryptor.XORKeyStream(frame, frame)
	copy(frame[:frameOffsetIV], copyFrame[:frameOffsetIV])

	obfs := &Obfuscated2{
		Decryptor: decryptor,
		Encryptor: encryptor,
	}

	return obfs, frame
}

func makeStreamCipher(key, iv []byte) cipher.Stream {
	block, _ := aes.NewCipher(key) // nolint: gosec
	return cipher.NewCTR(block, iv)
}
