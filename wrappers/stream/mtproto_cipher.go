package stream

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"net"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/mtproto/rpc"
	"github.com/9seconds/mtg/utils"
)

type mtprotoCipherPurpose uint8

const (
	mtprotoCipherPurposeClient mtprotoCipherPurpose = iota
	mtprotoCipherPurposeServer
)

var mtprotoEmptyIP = [4]byte{0x00, 0x00, 0x00, 0x00}

func NewMiddleProxyCipher(parent conntypes.StreamReadWriteCloser,
	req *rpc.NonceRequest,
	resp *rpc.NonceResponse,
	secret []byte) conntypes.StreamReadWriteCloser {
	localAddr := parent.LocalAddr()
	remoteAddr := parent.RemoteAddr()

	encKey, encIV := mtprotoDeriveKeys(mtprotoCipherPurposeClient,
		req,
		resp,
		localAddr,
		remoteAddr,
		secret)
	decKey, decIV := mtprotoDeriveKeys(mtprotoCipherPurposeServer,
		req,
		resp,
		localAddr,
		remoteAddr,
		secret)

	enc, _ := mtprotoMakeEncrypterDecrypter(encKey, encIV)
	_, dec := mtprotoMakeEncrypterDecrypter(decKey, decIV)

	return newBlockCipher(parent, enc, dec)
}

func mtprotoDeriveKeys(purpose mtprotoCipherPurpose,
	req *rpc.NonceRequest,
	resp *rpc.NonceResponse,
	client, remote *net.TCPAddr,
	secret []byte) ([]byte, []byte) {
	message := bytes.Buffer{}

	message.Write(resp.Nonce)
	message.Write(req.Nonce)
	message.Write(req.CryptoTS)

	clientIPv4 := mtprotoEmptyIP[:]
	serverIPv4 := mtprotoEmptyIP[:]

	if client.IP.To4() != nil {
		clientIPv4 = utils.ReverseBytes(client.IP.To4())
		serverIPv4 = utils.ReverseBytes(remote.IP.To4())
	}

	message.Write(serverIPv4)

	var port [2]byte

	binary.LittleEndian.PutUint16(port[:], uint16(client.Port))
	message.Write(port[:])

	switch purpose {
	case mtprotoCipherPurposeClient:
		message.WriteString("CLIENT")
	case mtprotoCipherPurposeServer:
		message.WriteString("SERVER")
	default:
		panic("Unexpected cipher purpose")
	}

	message.Write(clientIPv4)
	binary.LittleEndian.PutUint16(port[:], uint16(remote.Port))
	message.Write(port[:])
	message.Write(secret)
	message.Write(resp.Nonce)

	if client.IP.To4() == nil {
		message.Write(client.IP.To16())
		message.Write(remote.IP.To16())
	}

	message.Write(req.Nonce)

	data := message.Bytes()
	md5sum := md5.Sum(data[1:]) // nolint: gas
	sha1sum := sha1.Sum(data)   // nolint: gosec

	key := append(md5sum[:12], sha1sum[:]...)
	iv := md5.Sum(data[2:]) // nolint: gas

	return key, iv[:]
}

func mtprotoMakeEncrypterDecrypter(key, iv []byte) (cipher.BlockMode, cipher.BlockMode) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	return cipher.NewCBCEncrypter(block, iv), cipher.NewCBCDecrypter(block, iv)
}
