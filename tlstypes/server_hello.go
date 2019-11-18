package tlstypes

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"io"
	mrand "math/rand"

	"golang.org/x/crypto/curve25519"

	"mtg/config"
)

type ServerHello struct {
	Handshake

	clientHello *ClientHello
}

func (s ServerHello) WelcomePacket() []byte {
	s.Random = [32]byte{}
	rec := Record{
		Type:    RecordTypeHandshake,
		Version: Version12,
		Data:    &s,
	}
	buf := bytes.NewBuffer(rec.Bytes())

	recChangeCipher := Record{
		Type:    RecordTypeChangeCipherSpec,
		Version: Version12,
		Data:    RawBytes([]byte{0x01}),
	}
	buf.Write(recChangeCipher.Bytes())

	hostCert := make([]byte, 1024+mrand.Intn(3092))
	rand.Read(hostCert) // nolint: errcheck

	recData := Record{
		Type:    RecordTypeApplicationData,
		Version: Version12,
		Data:    RawBytes(hostCert),
	}
	buf.Write(recData.Bytes())
	packet := buf.Bytes()

	mac := hmac.New(sha256.New, config.C.Secret)
	mac.Write(s.clientHello.Random[:]) // nolint: errcheck
	mac.Write(packet)                  // nolint: errcheck
	copy(packet[11:], mac.Sum(nil))

	return packet
}

func NewServerHello(clientHello *ClientHello) *ServerHello {
	rv := &ServerHello{
		clientHello: clientHello,
	}

	rv.Type = HandshakeTypeServer
	rv.Version = Version12
	rv.SessionID = make([]byte, len(clientHello.SessionID))
	copy(rv.SessionID, clientHello.SessionID)

	tail := bytes.NewBuffer(CipherSuiteType_TLS_AES_128_GCM_SHA256_Bytes)
	tail.WriteByte(0x00) // no compression
	makeTLSExtensions(tail)
	rv.Tail = RawBytes(tail.Bytes())

	return rv
}

func makeTLSExtensions(buf io.Writer) {
	buf.Write([]byte{ // nolint: errcheck
		0x00, 0x2e, // 46 bytes of data
		0x00, 0x33, // Extension - Key Share
		0x00, 0x24, // 36 bytes
		0x00, 0x1d, // x25519 curve
		0x00, 0x20, // 32 bytes of key
	})

	var scalar [32]byte

	rand.Read(scalar[:]) // nolint: errcheck
	curve, _ := curve25519.X25519(scalar[:], curve25519.Basepoint)
	buf.Write(curve) // nolint: errcheck

	buf.Write([]byte{ // nolint: errcheck
		0x00, 0x2b, // Extension - Supported Versions
		0x00, 0x02, // 2 bytes are following
		0x03, 0x04, // TLS 1.3
	})
}
