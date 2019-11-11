package tlstypes

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"

	"mtg/config"
	"mtg/utils"
)

type ClientHello struct {
	Handshake
}

func (c ClientHello) Digest() []byte {
	dirtyDigest := c.Random
	c.Random = [32]byte{}

	rec := Record{
		Type:    RecordTypeHandshake,
		Version: Version10,
		Data:    &c,
	}

	mac := hmac.New(sha256.New, config.C.Secret)
	mac.Write(rec.Bytes()) // nolint: errcheck
	computedDigest := mac.Sum(nil)

	for i := range computedDigest {
		computedDigest[i] ^= dirtyDigest[i]
	}

	return computedDigest
}

func ParseClientHello(raw []byte) (*ClientHello, error) {
	rv := &ClientHello{}

	rv.Type = HandshakeType(raw[0])
	if rv.Type != HandshakeTypeClient {
		return nil, fmt.Errorf("incorrect handshake type %v", rv.Type)
	}

	raw = raw[1:]
	sizeUint24 := utils.Uint24{}
	copy(sizeUint24[:], utils.ReverseBytes(raw[:3]))
	size := int(utils.FromUint24(sizeUint24))

	raw = raw[3:]
	if len(raw) != size {
		return nil, fmt.Errorf("payload size mismatch (%d != %d)", len(raw), size)
	}

	versionRaw := raw[:2]

	switch {
	case bytes.Equal(versionRaw, Version13Bytes):
		rv.Version = Version13
	case bytes.Equal(versionRaw, Version12Bytes):
		rv.Version = Version12
	case bytes.Equal(versionRaw, Version11Bytes):
		rv.Version = Version11
	case bytes.Equal(versionRaw, Version10Bytes):
		rv.Version = Version10
	default:
		return nil, fmt.Errorf("unknown protocol version %v", versionRaw)
	}

	raw = raw[2:]
	copy(rv.Random[:], raw[:32])
	raw = raw[32:]

	sessionIDLength := int(raw[0])
	raw = raw[1:]
	rv.SessionID = make([]byte, sessionIDLength)
	copy(rv.SessionID, raw)
	raw = raw[sessionIDLength:]

	tail := make([]byte, len(raw))
	copy(tail, raw)
	rv.Tail = RawBytes(tail)

	return rv, nil
}
