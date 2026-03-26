package fake

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"io"
	rnd "math/rand/v2"

	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
	"golang.org/x/crypto/curve25519"
)

const (
	TypeHandshakeServer = 0x02
	ChangeCipherValue   = 0x01

	EllipticCurveLen = 32
)

var serverHelloSuffix = []byte{
	0x00,       // no compression
	0x00, 0x2e, // 46 bytes of data
	0x00, 0x2b, // Extension - Supported Versions
	0x00, 0x02, // 2 bytes are following
	0x03, 0x04, // TLS 1.3
	0x00, 0x33, // Extension - Key Share
	0x00, 0x24, // 36 bytes
	0x00, 0x1d, // x25519 curve
	0x00, 0x20, // 32 bytes of key
}

// NoiseParams controls the size of the fake ApplicationData record in ServerHello.
// If Mean is 0, the legacy random range (2500-4700) is used.
type NoiseParams struct {
	Mean   int
	Jitter int
}

func SendServerHello(w io.Writer, secret []byte, clientHello *ClientHello, noise NoiseParams) error {
	buf := &bytes.Buffer{}
	buf.Grow(tls.MaxRecordSize)

	generateServerHello(buf, clientHello)
	generateChangeCipherValue(buf)
	generateNoise(buf, noise)

	packet := buf.Bytes()
	digest := hmac.New(sha256.New, secret)

	digest.Write(clientHello.Random[:])
	digest.Write(packet)
	copy(packet[RandomOffset:], digest.Sum(nil))

	_, err := w.Write(packet)

	return err
}

func generateServerHello(buf *bytes.Buffer, hello *ClientHello) {
	payload := acquireBuffer()
	defer releaseBuffer(payload)

	generateServerHelloPayload(payload, hello)

	// 16 - type is 0x16 (handshake record)
	// 03 03 - legacy protocol version of "3,3" (TLS 1.2)
	// 00 7a - 0x7A (122) bytes of handshake message follows

	// 16 - type is 0x16 (handshake record)
	buf.WriteByte(tls.TypeHandshake)
	// 03 03 - legacy protocol version of "3,3" (TLS 1.2)
	buf.Write(tls.TLSVersion[:])
	// 00 7a - 0x7A (122) bytes of handshake message follows
	binary.Write(buf, binary.BigEndian, uint16(payload.Len())) //nolint: errcheck

	payload.WriteTo(buf) //nolint: errcheck
}

func generateServerHelloPayload(buf *bytes.Buffer, hello *ClientHello) {
	data := [4]byte{}

	payload := acquireBuffer()
	defer releaseBuffer(payload)

	generateServerHelloHandshakePayload(payload, hello)

	// 02 - handshake message type 0x02 (server hello)
	// 00 00 76 - 0x76 (118) bytes of server hello data follows
	buf.WriteByte(TypeHandshakeServer)
	// 00 00 76 - 0x76 (118) bytes of server hello data follows
	binary.BigEndian.PutUint32(data[:], uint32(payload.Len()))
	buf.Write(data[1:])

	payload.WriteTo(buf) //nolint: errcheck
}

func generateServerHelloHandshakePayload(buf *bytes.Buffer, hello *ClientHello) {
	//  The unusual version number ("3,3" representing TLS 1.2) is due to
	// TLS 1.0 being a minor revision of the SSL 3.0 protocol. Therefore
	// TLS 1.0 is represented by "3,1", TLS 1.1 is "3,2", and so on.
	buf.Write(tls.TLSVersion[:])

	buf.Write(emptyRandom[:])

	// 20 - 0x20 (32) bytes of session ID follow
	// e0 e1 ... fe ff - session ID copied from Client Hello
	buf.WriteByte(byte(len(hello.SessionID)))
	buf.Write(hello.SessionID)

	binary.Write(buf, binary.BigEndian, hello.CipherSuite) //nolint: errcheck

	buf.Write(serverHelloSuffix)

	scalar := [EllipticCurveLen]byte{}

	if _, err := rand.Read(scalar[:]); err != nil {
		panic(err)
	}

	curve, _ := curve25519.X25519(scalar[:], curve25519.Basepoint)
	buf.Write(curve)
}

func generateChangeCipherValue(buf *bytes.Buffer) {
	buf.WriteByte(tls.TypeChangeCipherSpec)
	buf.Write(tls.TLSVersion[:])
	binary.Write(buf, binary.BigEndian, uint16(1)) //nolint: errcheck
	buf.WriteByte(ChangeCipherValue)
}

// generateNoise writes a single ApplicationData record mimicking the combined
// size of a real TLS 1.3 encrypted server handshake (EncryptedExtensions +
// Certificate chain + CertificateVerify + Finished ≈ 2800-5000 bytes).
//
// NOTE: Must be exactly ONE ApplicationData record — the Telegram client reads
// ServerHello + CCS + 1 ApplicationData and computes HMAC over all three.
// Multiple records would cause HMAC mismatch and connection failure.
func generateNoise(buf *bytes.Buffer, noise NoiseParams) {
	var size int

	if noise.Mean > 0 && noise.Jitter > 0 {
		// Calibrated: use measured cert chain size ± jitter.
		size = noise.Mean - noise.Jitter + rnd.IntN(2*noise.Jitter)
		if size < 1000 {
			size = 1000
		}
	} else {
		// Legacy fallback: random in 2500-4700 range.
		size = 2500 + rnd.IntN(2200)
	}

	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}

	tls.WriteRecord(buf, data) //nolint: errcheck
}
