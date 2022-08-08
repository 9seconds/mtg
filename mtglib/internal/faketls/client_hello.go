package faketls

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/internal/faketls/record"
)

type ClientHello struct {
	Time        time.Time
	Random      [RandomLen]byte
	SessionID   []byte
	Host        string
	CipherSuite uint16
}

func (c ClientHello) Valid(hostname string, tolerateTimeSkewness time.Duration) error {
	if c.Host != "" && c.Host != hostname {
		return fmt.Errorf("incorrect hostname %s", hostname)
	}

	now := time.Now()

	timeDiff := now.Sub(c.Time)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	if timeDiff > tolerateTimeSkewness {
		return fmt.Errorf("incorrect timestamp. got=%d, now=%d, diff=%s",
			c.Time.Unix(), now.Unix(), timeDiff.String())
	}

	return nil
}

func ParseClientHello(secret, handshake []byte) (ClientHello, error) {
	hello := ClientHello{}

	if len(handshake) < ClientHelloMinLen {
		return hello, fmt.Errorf("lengh of handshake is too small: %d", len(handshake))
	}

	if handshake[0] != HandshakeTypeClient {
		return hello, fmt.Errorf("unknown handshake type %#x", handshake[0])
	}

	handshakeSizeBytes := [4]byte{0, handshake[1], handshake[2], handshake[3]}
	handshakeLength := binary.BigEndian.Uint32(handshakeSizeBytes[:])

	if len(handshake)-4 != int(handshakeLength) {
		return hello,
			fmt.Errorf("incorrect handshake size. manifested=%d, real=%d",
				handshakeLength, len(handshake)-4) //nolint: gomnd
	}

	copy(hello.Random[:], handshake[ClientHelloRandomOffset:])
	copy(handshake[ClientHelloRandomOffset:], clientHelloEmptyRandom)

	rec := record.AcquireRecord()
	defer record.ReleaseRecord(rec)

	rec.Type = record.TypeHandshake
	rec.Version = record.Version10
	rec.Payload.Write(handshake)

	// mac is calculated for the whole record, not only
	// for the payload part
	mac := hmac.New(sha256.New, secret)
	rec.Dump(mac) //nolint: errcheck

	computedRandom := mac.Sum(nil)

	for i := 0; i < RandomLen; i++ {
		computedRandom[i] ^= hello.Random[i]
	}

	if subtle.ConstantTimeCompare(clientHelloEmptyRandom[:RandomLen-4], computedRandom[:RandomLen-4]) != 1 {
		return hello, ErrBadDigest
	}

	timestamp := int64(binary.LittleEndian.Uint32(computedRandom[RandomLen-4:]))
	hello.Time = time.Unix(timestamp, 0)

	parseSessionID(&hello, handshake)
	parseCipherSuite(&hello, handshake)
	parseSNI(&hello, handshake)

	return hello, nil
}

func parseSessionID(hello *ClientHello, handshake []byte) {
	hello.SessionID = make([]byte, handshake[ClientHelloSessionIDOffset])
	copy(hello.SessionID, handshake[ClientHelloSessionIDOffset+1:])
}

func parseCipherSuite(hello *ClientHello, handshake []byte) {
	cipherSuiteOffset := ClientHelloSessionIDOffset + len(hello.SessionID) + 3 //nolint: gomnd
	hello.CipherSuite = binary.BigEndian.Uint16(handshake[cipherSuiteOffset : cipherSuiteOffset+2])
}

func parseSNI(hello *ClientHello, handshake []byte) {
	cipherSuiteOffset := ClientHelloSessionIDOffset + len(hello.SessionID) + 1
	handshake = handshake[cipherSuiteOffset:]

	cipherSuiteLength := binary.BigEndian.Uint16(handshake[:2])
	handshake = handshake[2+cipherSuiteLength:]

	compressionMethodsLength := int(handshake[0])
	handshake = handshake[1+compressionMethodsLength:]

	extensionsLength := binary.BigEndian.Uint16(handshake[:2])
	handshake = handshake[2 : 2+extensionsLength]

	for len(handshake) > 0 {
		if binary.BigEndian.Uint16(handshake[:2]) != ExtensionSNI {
			extensionsLength := binary.BigEndian.Uint16(handshake[2:4])
			handshake = handshake[4+extensionsLength:]

			continue
		}

		hostnameLength := binary.BigEndian.Uint16(handshake[7:9])
		handshake = handshake[9:]
		hello.Host = string(handshake[:int(hostnameLength)])

		return
	}
}
