package faketls

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"io"
	mrand "math/rand"

	"github.com/9seconds/mtg/v2/mtglib/internal/faketls/record"
	"golang.org/x/crypto/curve25519"
)

func SendWelcomePacket(writer io.Writer, secret []byte, clientHello ClientHello) error {
	buf := acquireBytesBuffer()
	defer releaseBytesBuffer(buf)

	rec := record.AcquireRecord()
	defer record.ReleaseRecord(rec)

	rec.Type = record.TypeHandshake
	rec.Version = record.Version12

	generateServerHello(&rec.Payload, clientHello)
	rec.Dump(buf) //nolint: errcheck
	rec.Reset()

	rec.Type = record.TypeChangeCipherSpec
	rec.Version = record.Version12
	rec.Payload.WriteByte(ChangeCipherValue)

	rec.Dump(buf) //nolint: errcheck
	rec.Reset()

	rec.Type = record.TypeApplicationData
	rec.Version = record.Version12

	if _, err := io.CopyN(&rec.Payload, rand.Reader, int64(1024+mrand.Intn(3092))); err != nil { //nolint: gomnd
		panic(err)
	}

	rec.Dump(buf) //nolint: errcheck

	packet := buf.Bytes()
	mac := hmac.New(sha256.New, secret)

	mac.Write(clientHello.Random[:])
	mac.Write(packet)

	copy(packet[WelcomePacketRandomOffset:], mac.Sum(nil))

	if _, err := writer.Write(packet); err != nil {
		return err //nolint: wrapcheck
	}

	return nil
}

func generateServerHello(writer io.Writer, clientHello ClientHello) {
	bodyBuf := acquireBytesBuffer()
	defer releaseBytesBuffer(bodyBuf)

	sliceBuf := [2]byte{}
	digest := [RandomLen]byte{}

	binary.BigEndian.PutUint16(sliceBuf[:], uint16(record.Version12))
	bodyBuf.Write(sliceBuf[:])
	bodyBuf.Write(digest[:])
	bodyBuf.WriteByte(byte(len(clientHello.SessionID)))
	bodyBuf.Write(clientHello.SessionID)

	binary.BigEndian.PutUint16(sliceBuf[:], clientHello.CipherSuite)
	bodyBuf.Write(sliceBuf[:])
	bodyBuf.Write(serverHelloSuffix)

	scalar := [32]byte{}

	if _, err := rand.Read(scalar[:]); err != nil {
		panic(err)
	}

	curve, _ := curve25519.X25519(scalar[:], curve25519.Basepoint)
	bodyBuf.Write(curve)

	header := [4]byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(header[:], uint32(bodyBuf.Len()))
	header[0] = HandshakeTypeServer

	writer.Write(header[:]) //nolint: errcheck
	bodyBuf.WriteTo(writer) //nolint: errcheck
}
