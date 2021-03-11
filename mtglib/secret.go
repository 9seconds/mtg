package mtglib

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	SecretKeyLength = 16

	secretFakeTLSFirstByte byte = 238
)

var secretEmptyKey [SecretKeyLength]byte

type Secret struct {
	Key  [SecretKeyLength]byte
	Host string
}

func (s Secret) MarshalText() ([]byte, error) {
	if s.Key == secretEmptyKey {
		return nil, nil
	}

	return []byte(s.String()), nil
}

func (s *Secret) UnmarshalText(data []byte) error {
	text := string(data)
	if text == "" {
		return ErrSecretEmpty
	}

	var (
		decoded []byte
		err     error
	)

	if strings.HasPrefix(text, "ee") {
		decoded, err = hex.DecodeString(strings.TrimPrefix(text, "ee"))
	}

	if err != nil || len(decoded) <= SecretKeyLength {
		decoded, err = base64.RawURLEncoding.DecodeString(text)

		if err != nil {
			return fmt.Errorf("incorrect secret format: %w", err)
		}

		if len(decoded) <= SecretKeyLength {
			return fmt.Errorf("secret has incorrect length %d", len(text))
		}

		if decoded[0] != secretFakeTLSFirstByte {
			return fmt.Errorf("incorrect first byte: %v", decoded[0])
		}

		decoded = decoded[1:]
	}

	copy(s.Key[:], decoded[:SecretKeyLength])
	s.Host = string(decoded[SecretKeyLength:])

	if s.Host == "" {
		return fmt.Errorf("hostname cannot be empty: %s", text)
	}

	return nil
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
