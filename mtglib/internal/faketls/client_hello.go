package faketls

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/internal/faketls/record"
)

type ClientHello struct {
	Time        time.Time
	Random      [RandomLen]byte
	SessionID   []byte
	CipherSuite uint16
}

func ParseClientHello(secret, handshake []byte) (ClientHello, error) {
	hello := ClientHello{}

	if len(handshake) < ClientHelloMinLen {
		return hello, fmt.Errorf("lengh of handshake is too small: %d", len(handshake))
	}

	if handshake[0] != HandshakeTypeClient {
		return hello, fmt.Errorf("unknown handshake type %#x", handshake[0])
	}

	copy(hello.Random[:], handshake[ClientHelloRandomOffset:])

	for i := ClientHelloRandomOffset; i < ClientHelloRandomOffset+RandomLen; i++ {
		handshake[i] = 0
	}

	rec := record.AcquireRecord()
	defer record.ReleaseRecord(rec)

	rec.Type = record.TypeHandshake
	rec.Version = record.Version10
	rec.Payload.Write(handshake)

	// mac is calculated for the whole record, not only
	// for the payload part
	mac := hmac.New(sha256.New, secret)
	rec.Dump(mac) // nolint: errcheck

	computedRandom := mac.Sum(nil)

	for i := 0; i < RandomLen; i++ {
		computedRandom[i] ^= hello.Random[i]
	}

	for i := 0; i < RandomLen-4; i++ {
		if computedRandom[i] != 0 {
			return hello, ErrBadDigest
		}
	}

	timestamp := int64(binary.LittleEndian.Uint32(computedRandom[RandomLen-4:]))
	hello.Time = time.Unix(timestamp, 0)

	hello.SessionID = make([]byte, handshake[ClientHelloSessionIDOffset])
	copy(hello.SessionID, handshake[ClientHelloSessionIDOffset+1:])

	cipherSuiteOffset := ClientHelloSessionIDOffset + len(hello.SessionID) + 3 // nolint: gomnd
	hello.CipherSuite = binary.BigEndian.Uint16(handshake[cipherSuiteOffset : cipherSuiteOffset+2])

	return hello, nil
}
