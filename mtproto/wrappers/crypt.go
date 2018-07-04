package wrappers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"net"

	"github.com/9seconds/mtg/mtproto/rpc"
	"github.com/9seconds/mtg/wrappers"
)

type CipherPurpose uint8

const (
	CipherPurposeClient CipherPurpose = iota
	CipherPurposeServer
)

var emptyIP = [4]byte{0x00, 0x00, 0x00, 0x00}

func NewMiddleProxyCipherRWC(conn wrappers.ReadWriteCloserWithAddr, req *rpc.RPCNonceRequest, resp *rpc.RPCNonceResponse, secret []byte) wrappers.ReadWriteCloserWithAddr {
	localAddr := conn.LocalAddr()
	remoteAddr := conn.RemoteAddr()

	encKey, encIV := makeKeys(CipherPurposeClient, req, resp, localAddr, remoteAddr, secret)
	decKey, decIV := makeKeys(CipherPurposeServer, req, resp, localAddr, remoteAddr, secret)

	enc, _ := makeEncrypterDecrypter(encKey, encIV)
	_, dec := makeEncrypterDecrypter(decKey, decIV)

	return wrappers.NewBlockCipherRWC(conn, enc, dec)
}

func makeKeys(purpose CipherPurpose, req *rpc.RPCNonceRequest, resp *rpc.RPCNonceResponse,
	client *net.TCPAddr, remote *net.TCPAddr, secret []byte) ([]byte, []byte) {
	message := bytes.Buffer{}
	message.Write(resp.Nonce[:])
	message.Write(req.Nonce[:])
	message.Write(req.CryptoTS[:])

	clientIPv4 := emptyIP[:]
	serverIPv4 := emptyIP[:]
	if client.IP.To4() != nil {
		clientIPv4 = reverseBytes(client.IP.To4())
		serverIPv4 = reverseBytes(remote.IP.To4())
	}
	message.Write(serverIPv4)

	var port [2]byte
	binary.LittleEndian.PutUint16(port[:], uint16(client.Port))
	message.Write(port[:])

	switch purpose {
	case CipherPurposeClient:
		message.WriteString("CLIENT")
	case CipherPurposeServer:
		message.WriteString("SERVER")
	default:
		panic("Unexpected cipher purpose")
	}

	message.Write(clientIPv4)
	binary.LittleEndian.PutUint16(port[:], uint16(remote.Port))
	message.Write(port[:])
	message.Write(secret)
	message.Write(resp.Nonce[:])

	if client.IP.To4() == nil {
		message.Write(client.IP.To16())
		message.Write(remote.IP.To16())
	}
	message.Write(req.Nonce[:])

	data := message.Bytes()
	md5sum := md5.Sum(data[1:])
	sha1sum := sha1.Sum(data)

	key := append(md5sum[:12], sha1sum[:]...)
	iv := md5.Sum(data[2:])

	return key, iv[:]
}

func makeEncrypterDecrypter(key, iv []byte) (cipher.BlockMode, cipher.BlockMode) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	return cipher.NewCBCEncrypter(block, iv), cipher.NewCBCDecrypter(block, iv)
}

func reverseBytes(data []byte) []byte {
	rv := make([]byte, len(data))
	for k, v := range data {
		rv[len(data)-1-k] = v
	}

	return rv
}
