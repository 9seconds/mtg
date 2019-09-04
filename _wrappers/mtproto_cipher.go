package wrappers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"  // nolint: gas
	"crypto/sha1" // nolint: gosec
	"encoding/binary"
	"net"

	"github.com/9seconds/mtg/mtproto/rpc"
	"github.com/9seconds/mtg/utils"
)

type cipherPurpose uint8

const (
	cipherPurposeClient cipherPurpose = iota
	cipherPurposeServer
)

var emptyIP = [4]byte{0x00, 0x00, 0x00, 0x00}

// NewMiddleProxyCipher creates new block cipher to proxy<->telegram
// connection.
func NewMiddleProxyCipher(conn StreamReadWriteCloser,
	req *rpc.NonceRequest, resp *rpc.NonceResponse, secret []byte) StreamReadWriteCloser {
	localAddr := conn.LocalAddr()
	remoteAddr := conn.RemoteAddr()

	encKey, encIV := deriveKeys(cipherPurposeClient, req, resp, localAddr, remoteAddr, secret)
	decKey, decIV := deriveKeys(cipherPurposeServer, req, resp, localAddr, remoteAddr, secret)

	enc, _ := makeEncrypterDecrypter(encKey, encIV)
	_, dec := makeEncrypterDecrypter(decKey, decIV)

	return NewBlockCipher(conn, enc, dec)
}

func deriveKeys(purpose cipherPurpose, req *rpc.NonceRequest, resp *rpc.NonceResponse,
	client, remote *net.TCPAddr, secret []byte) ([]byte, []byte) {
	message := bytes.Buffer{}
	message.Write(resp.Nonce)   // nolint: gosec
	message.Write(req.Nonce)    // nolint: gosec
	message.Write(req.CryptoTS) // nolint: gosec

	clientIPv4 := emptyIP[:]
	serverIPv4 := emptyIP[:]
	if client.IP.To4() != nil {
		clientIPv4 = utils.ReverseBytes(client.IP.To4())
		serverIPv4 = utils.ReverseBytes(remote.IP.To4())
	}
	message.Write(serverIPv4) // nolint: gosec

	var port [2]byte
	binary.LittleEndian.PutUint16(port[:], uint16(client.Port))
	message.Write(port[:]) // nolint: gosec

	switch purpose {
	case cipherPurposeClient:
		message.WriteString("CLIENT") // nolint: gosec
	case cipherPurposeServer:
		message.WriteString("SERVER") // nolint: gosec
	default:
		panic("Unexpected cipher purpose")
	}

	message.Write(clientIPv4) // nolint: gosec
	binary.LittleEndian.PutUint16(port[:], uint16(remote.Port))
	message.Write(port[:])    // nolint: gosec
	message.Write(secret)     // nolint: gosec
	message.Write(resp.Nonce) // nolint: gosec

	if client.IP.To4() == nil {
		message.Write(client.IP.To16()) // nolint: gosec
		message.Write(remote.IP.To16()) // nolint: gosec
	}
	message.Write(req.Nonce) // nolint: gosec

	data := message.Bytes()
	md5sum := md5.Sum(data[1:]) // nolint: gas
	sha1sum := sha1.Sum(data)   // nolint: gosec

	key := append(md5sum[:12], sha1sum[:]...)
	iv := md5.Sum(data[2:]) // nolint: gas

	return key, iv[:]
}

func makeEncrypterDecrypter(key, iv []byte) (cipher.BlockMode, cipher.BlockMode) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	return cipher.NewCBCEncrypter(block, iv), cipher.NewCBCDecrypter(block, iv)
}
