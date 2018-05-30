package obfuscated2

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"

	"github.com/juju/errors"
)

type Obfuscated2 struct {
	decryptor cipher.Stream
	encryptor cipher.Stream
}

func (o *Obfuscated2) Encrypt(data []byte) []byte {
	buf := make([]byte, len(data))
	o.encryptor.XORKeyStream(buf, data)
	return buf
}

func (o *Obfuscated2) Decrypt(data []byte) []byte {
	buf := make([]byte, len(data))
	o.decryptor.XORKeyStream(buf, data)
	return buf
}

func ParseObfuscated2ClientFrame(secret, data []byte) (*Obfuscated2, int16, error) {
	frame := Frame(data)

	decHasher := sha256.New()
	decHasher.Write(frame.Key())
	decHasher.Write(secret)
	decryptor := makeStreamCipher(decHasher.Sum(nil), frame.IV())

	invertedFrame := frame.Invert()
	encHasher := sha256.New()
	encHasher.Write(invertedFrame.Key())
	encHasher.Write(secret)
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
