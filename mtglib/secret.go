package mtglib

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const secretFakeTLSFirstByte byte = 0xee

var secretEmptyKey [SecretKeyLength]byte

// Secret is a data structure that presents a secret.
//
// Telegram secret is not a simple string like
// "ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d".
// Actually, this is a serialized datastructure of 2 parts: key and host.
//
//	ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d
//	|-|-------------------------------|-------------------------------------------
//	p key                             hostname
//
// Serialized secret starts with 'ee'. Actually, in the past we also had 'dd'
// secrets and prefixless ones. But this is history. Currently, we do have only
// 'ee' secrets which mean faketls + protection from statistical attacks on a
// length. 'ee' is a byte 238 (0xee).
//
// After that, we have 16 bytes of the key. This is a random generated secret
// data of the proxy and this data is used to derive authentication schemas.
// These secrets are mixed into hmacs and sha256 checksums which are used to
// build AEAD ciphers for obfuscated2 protocol and ensure faketls handshake.
//
// Host is a domain fronting hostname in latin1 (ASCII) encoding. This hostname
// should be used for SNI in faketls and MTG verifies it. Also, this is when
// mtg gets about a domain fronting hostname.
//
// Secrets can be serialized into 2 forms: hex and base64. If you decode both
// forms into bytes, you'll get the same byte array. Telegram clients nowadays
// accept all forms.
type Secret struct {
	// Key is a set of bytes used for traffic authentication.
	Key [SecretKeyLength]byte

	// Host is a domain fronting hostname.
	Host string
}

// MarshalText is to support text.Marshaller interface.
func (s Secret) MarshalText() ([]byte, error) {
	if s.Valid() {
		return []byte(s.String()), nil
	}

	return nil, nil
}

// UnmarshalText is to support text.Unmarshaller interface.
func (s *Secret) UnmarshalText(data []byte) error {
	return s.Set(string(data))
}

func (s *Secret) Set(text string) error {
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

	if len(decoded) < 2 { //nolint: gomnd // we need at least 1 byte here
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

// Valid checks if this secret is valid and can be used in proxy.
func (s Secret) Valid() bool {
	return s.Key != secretEmptyKey && s.Host != ""
}

// String is to support fmt.Stringer interface.
func (s Secret) String() string {
	return s.Base64()
}

// Base64 returns a base64-encoded form of this secret.
func (s Secret) Base64() string {
	return base64.RawURLEncoding.EncodeToString(s.makeBytes())
}

// Hex returns a hex-encoded form of this secret (ee-secret).
func (s Secret) Hex() string {
	return hex.EncodeToString(s.makeBytes())
}

func (s *Secret) makeBytes() []byte {
	data := append([]byte{secretFakeTLSFirstByte}, s.Key[:]...)
	data = append(data, s.Host...)

	return data
}

// GenerateSecret makes a new secret with a given hostname.
func GenerateSecret(hostname string) Secret {
	s := Secret{
		Host: hostname,
	}

	if _, err := rand.Read(s.Key[:]); err != nil {
		panic(err)
	}

	return s
}

// ParseSecret parses a secret (both hex and base64 forms).
func ParseSecret(secret string) (Secret, error) {
	s := Secret{}

	return s, s.UnmarshalText([]byte(secret))
}
