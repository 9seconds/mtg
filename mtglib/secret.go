package mtglib

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

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

func (s *Secret) UnmarshalText(text []byte) error {
	sc, err := ParseSecret(string(text))
	if err != nil {
		return err
	}

	*s = sc

	return nil
}

func (s Secret) Base64() string {
	return s.String()
}

func (s Secret) EE() string {
	return "ee" + hex.EncodeToString(append(s.Key, s.Host...))
}

func (s Secret) String() string {
	return base64.StdEncoding.EncodeToString(append(s.Key, s.Host...))
}

func ParseSecret(secret string) (Secret, error) {
	rv := Secret{}

	if secret == "" {
		return rv, errors.New("secret cannot be empty")
	}

	decoded, err := base64.RawStdEncoding.DecodeString(secret)
	if err != nil && strings.HasPrefix(secret, "ee") {
		decoded, err = hex.DecodeString(strings.TrimPrefix(secret, "ee"))
	}

	if err != nil {
		return rv, fmt.Errorf("incorrect secret format: %w", err)
	}

	if len(decoded) < 33 {
		return rv, fmt.Errorf("secret %s has incorrect length", secret)
	}

	rv.Key = decoded[:32]
	rv.Host = string(decoded[32:])

	return rv, nil
}
