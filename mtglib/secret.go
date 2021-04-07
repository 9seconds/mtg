package mtglib

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const secretFakeTLSFirstByte byte = 0xee

var secretEmptyKey [SecretKeyLength]byte

type Secret struct {
	Key  [SecretKeyLength]byte
	Host string
}

func (s Secret) MarshalText() ([]byte, error) {
	if s.Valid() {
		return []byte(s.String()), nil
	}

	return nil, nil
}

func (s *Secret) UnmarshalText(data []byte) error {
	text := string(data)
	if text == "" {
		return ErrSecretEmpty
	}

	decoded, err := hex.DecodeString(text)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(text)
	}

	if err != nil {
		return fmt.Errorf("incorrect secret format: %w", err)
	}

	if len(decoded) < 2 { // nolint: gomnd // we need at least 1 byte here
		return fmt.Errorf("secret is truncated, length=%d", len(decoded))
	}

	if decoded[0] != secretFakeTLSFirstByte {
		return fmt.Errorf("incorrect first byte of secret: %#x", decoded[0])
	}

	decoded = decoded[1:]
	if len(decoded) < SecretKeyLength {
		return fmt.Errorf("secret has incorrect length %d", len(decoded))
	}

	copy(s.Key[:], decoded[:SecretKeyLength])
	s.Host = string(decoded[SecretKeyLength:])

	if s.Host == "" {
		return fmt.Errorf("hostname cannot be empty: %s", text)
	}

	return nil
}

func (s Secret) Valid() bool {
	return s.Key != secretEmptyKey && s.Host != ""
}

func (s Secret) String() string {
	return s.Base64()
}

func (s Secret) Base64() string {
	return base64.RawURLEncoding.EncodeToString(s.makeBytes())
}

func (s Secret) Hex() string {
	return hex.EncodeToString(s.makeBytes())
}

func (s *Secret) makeBytes() []byte {
	data := append([]byte{secretFakeTLSFirstByte}, s.Key[:]...)
	data = append(data, s.Host...)

	return data
}

func GenerateSecret(hostname string) Secret {
	s := Secret{
		Host: hostname,
	}

	if _, err := rand.Read(s.Key[:]); err != nil {
		panic(err)
	}

	return s
}

func ParseSecret(secret string) (Secret, error) {
	s := Secret{}

	return s, s.UnmarshalText([]byte(secret))
}
