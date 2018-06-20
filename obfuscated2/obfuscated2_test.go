package obfuscated2

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObfs2TelegramFrameDecrypt(t *testing.T) {
	_, frame := MakeTelegramObfuscated2Frame()
	decryptor := makeStreamCipher(frame.Key(), frame.IV())

	decrypted := make(Frame, FrameLen)
	decryptor.XORKeyStream(decrypted, *frame)

	assert.True(t, decrypted.Valid())
}

func TestObfs2TelegramDecryptEncryptDecrypt(t *testing.T) {
	obfs2, frame := MakeTelegramObfuscated2Frame()
	inverted := frame.Invert()
	encryptor := makeStreamCipher(inverted.Key(), inverted.IV())

	data := []byte{1, 2, 3}
	encrypted := make([]byte, 3)
	encryptor.XORKeyStream(encrypted, data)
	decrypted := make([]byte, 3)
	obfs2.Decryptor.XORKeyStream(decrypted, encrypted)

	assert.Equal(t, data, decrypted)
}

func TestObfs2Full(t *testing.T) {
	secret := []byte{1, 2, 3, 4, 5}

	clientFrame := generateFrame()
	clientHasher := sha256.New()
	clientHasher.Write(clientFrame.Key())
	clientHasher.Write(secret)
	clientKey := clientHasher.Sum(nil)

	encryptor := makeStreamCipher(clientKey, clientFrame.IV())
	encrypted := make(Frame, FrameLen)
	encryptor.XORKeyStream(encrypted, *clientFrame)
	copy(encrypted[:56], (*clientFrame)[:56])

	invertedClientFrame := clientFrame.Invert()
	clientHasher = sha256.New()
	clientHasher.Write(invertedClientFrame.Key())
	clientHasher.Write(secret)
	invertedClientKey := clientHasher.Sum(nil)
	clientDecryptor := makeStreamCipher(invertedClientKey, invertedClientFrame.IV())

	clientObfs, _, err := ParseObfuscated2ClientFrame(secret, &encrypted)
	assert.Nil(t, err)

	tgObfs, tgFrame := MakeTelegramObfuscated2Frame()
	tgDecryptor := makeStreamCipher(tgFrame.Key(), tgFrame.IV())
	decrypted := make(Frame, FrameLen)
	tgDecryptor.XORKeyStream(decrypted, *tgFrame)
	assert.True(t, decrypted.Valid())

	tgInvertedFrame := tgFrame.Invert()
	tgEncryptor := makeStreamCipher(tgInvertedFrame.Key(), tgInvertedFrame.IV())

	message := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	tgEncryptedMessage := make([]byte, len(message))
	tgEncryptor.XORKeyStream(tgEncryptedMessage, message)

	tgEncDecryptedMessage := make([]byte, len(tgEncryptedMessage))
	tgObfs.Decryptor.XORKeyStream(tgEncDecryptedMessage, tgEncryptedMessage)
	assert.Equal(t, message, tgEncDecryptedMessage)

	clientEncryptedMessage := make([]byte, len(tgEncDecryptedMessage))
	clientObfs.Encryptor.XORKeyStream(clientEncryptedMessage, tgEncDecryptedMessage)
	finalMessage := make([]byte, len(clientEncryptedMessage))
	clientDecryptor.XORKeyStream(finalMessage, clientEncryptedMessage)

	assert.Equal(t, finalMessage, message)
}
