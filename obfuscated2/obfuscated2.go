package obfuscated2

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"

	"github.com/juju/errors"
)

// Obfuscated2 contains AES CTR encryption and decryption streams
// for telegram connection.
type Obfuscated2 struct {
	decryptor cipher.Stream
	encryptor cipher.Stream
}

// Encrypt encrypts given data.
func (o *Obfuscated2) Encrypt(data []byte) []byte {
	buf := make([]byte, len(data))
	o.encryptor.XORKeyStream(buf, data)
	return buf
}

// Decrypt decrypts given data.
func (o *Obfuscated2) Decrypt(data []byte) []byte {
	buf := make([]byte, len(data))
	o.decryptor.XORKeyStream(buf, data)
	return buf
}

// ParseObfuscated2ClientFrame parses client frame. Please check this link for
// details: http://telegra.ph/telegram-blocks-wtf-05-26
//
// Beware, link above is in russian.
func ParseObfuscated2ClientFrame(secret, data []byte) (*Obfuscated2, int16, error) {
	frame := Frame(data)

	decHasher := sha256.New()
	decHasher.Write(frame.Key()) // nolint: errcheck
	decHasher.Write(secret)      // nolint: errcheck
	decryptor := makeStreamCipher(decHasher.Sum(nil), frame.IV())

	invertedFrame := frame.Invert()
	encHasher := sha256.New()
	encHasher.Write(invertedFrame.Key()) // nolint: errcheck
	encHasher.Write(secret)              // nolint: errcheck
	encryptor := makeStreamCipher(encHasher.Sum(nil), invertedFrame.IV())

	decryptedFrame := make(Frame, FrameLen)
	decryptor.XORKeyStream(decryptedFrame, frame)
	if !decryptedFrame.Valid() {
		return nil, 0, errors.New("Unknown protocol")
	}

	obfs := &Obfuscated2{
		decryptor: decryptor,
		encryptor: encryptor,
	}

	return obfs, decryptedFrame.DC(), nil
}

// MakeTelegramObfuscated2Frame creates new handshake frame to send to
// Telegram.
// https://blog.susanka.eu/how-telegram-obfuscates-its-mtproto-traffic/
func MakeTelegramObfuscated2Frame() (*Obfuscated2, Frame) {
	frame := generateFrame()

	encryptor := makeStreamCipher(frame.Key(), frame.IV())
	decryptorFrame := frame.Invert()
	decryptor := makeStreamCipher(decryptorFrame.Key(), decryptorFrame.IV())

	copyFrame := make(Frame, frameOffsetIV)
	copy(copyFrame, frame)
	encryptor.XORKeyStream(frame, frame)
	copy(frame, copyFrame)

	obfs := &Obfuscated2{
		decryptor: decryptor,
		encryptor: encryptor,
	}

	return obfs, frame
}

func makeStreamCipher(key, iv []byte) cipher.Stream {
	block, _ := aes.NewCipher(key)
	return cipher.NewCTR(block, iv)
}
