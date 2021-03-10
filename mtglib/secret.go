package mtglib

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

const SecretKeyLength = 32

type Secret struct {
	Key  []byte
	Host string
}

func (s *Secret) MarshalText() ([]byte, error) {
	if s == nil {
		return nil, nil
	}

	return []byte(s.String()), nil
}

func (s *Secret) UnmarshalText(data []byte) error {
	text := string(data)

	if text == "" {
		return ErrSecretEmpty
	}

	decoded, err := base64.RawStdEncoding.DecodeString(text)
	if err != nil && strings.HasPrefix(text, "ee") {
		decoded, err = hex.DecodeString(strings.TrimPrefix(text, "ee"))
	}

	if err != nil {
		return fmt.Errorf("incorrect secret format: %w", err)
	}

	if len(decoded) <= SecretKeyLength {
		return fmt.Errorf("secret has incorrect length %d", len(text))
	}

	s.Key = decoded[:SecretKeyLength]
	s.Host = string(decoded[SecretKeyLength:])

	return nil
}

func (s Secret) Base64() string {
	return base64.StdEncoding.EncodeToString(append(s.Key[:], s.Host...))
}

func (s Secret) String() string {
	return s.Base64()
}

func (s Secret) EE() string {
	return "ee" + hex.EncodeToString(append(s.Key[:], s.Host...))
}

func GenerateSecret(hostname string) Secret {
	s := Secret{
		Key:  make([]byte, SecretKeyLength),
		Host: hostname,
	}

	if _, err := rand.Read(s.Key); err != nil {
		panic(err)
	}

	return s
}
