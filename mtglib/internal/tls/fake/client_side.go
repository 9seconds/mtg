package fake

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"slices"
	"time"
)

const (
	TypeHandshakeClient = 0x01

	RandomLen = 32
	// record_type(1) + version(2) + size(2) + handshake_type(1) + uint24_length(3) + client_version(2)
	RandomOffset = 1 + 2 + 2 + 1 + 3 + 2

	sniDNSNamesListType = 0
)

var (
	emptyRandom = [RandomLen]byte{}
	extTypeSNI  = [2]byte{}
)

type ClientHello struct {
	Random      [RandomLen]byte
	SessionID   []byte
	CipherSuite uint16
}

// ReadClientHelloResult contains the parsed ClientHello and the index of the
// secret that matched the HMAC validation.
type ReadClientHelloResult struct {
	Hello        *ClientHello
	MatchedIndex int
}

func ReadClientHello(
	conn net.Conn,
	secret []byte,
	hostname string,
	tolerateTimeSkewness time.Duration,
) (*ClientHello, error) {
	result, err := ReadClientHelloMulti(conn, [][]byte{secret}, hostname, tolerateTimeSkewness)
	if err != nil {
		return nil, err
	}

	return result.Hello, nil
}

// ReadClientHelloMulti is like ReadClientHello but accepts multiple secrets.
// It tries each secret until one validates the HMAC. On success it returns
// the ClientHello and the index of the matched secret.
func ReadClientHelloMulti(
	conn net.Conn,
	secrets [][]byte,
	hostname string,
	tolerateTimeSkewness time.Duration,
) (*ReadClientHelloResult, error) {
	if err := conn.SetReadDeadline(time.Now().Add(ClientHelloReadTimeout)); err != nil {
		return nil, fmt.Errorf("cannot set read deadline: %w", err)
	}
	defer conn.SetReadDeadline(resetDeadline) //nolint: errcheck

	clientHelloCopy, handshakeReader, err := parseClientHello(conn)
	if err != nil {
		return nil, fmt.Errorf("cannot read client hello: %w", err)
	}

	hello, err := parseHandshake(handshakeReader)
	if err != nil {
		return nil, fmt.Errorf("cannot parse handshake: %w", err)
	}

	sniHostnames, err := parseSNI(handshakeReader)
	if err != nil {
		return nil, fmt.Errorf("cannot parse SNI: %w", err)
	}

	if !slices.Contains(sniHostnames, hostname) {
		return nil, fmt.Errorf("cannot find %s in %v", hostname, sniHostnames)
	}

	// Save the full client hello bytes so we can replay them for each secret.
	handshakeBytes := clientHelloCopy.Bytes()

	for idx, secret := range secrets {
		digest := hmac.New(sha256.New, secret)

		// Write the handshake with client random all nullified.
		digest.Write(handshakeBytes[:RandomOffset])
		digest.Write(emptyRandom[:])
		digest.Write(handshakeBytes[RandomOffset+RandomLen:])

		computed := digest.Sum(nil)

		for i := range RandomLen {
			computed[i] ^= hello.Random[i]
		}

		if subtle.ConstantTimeCompare(emptyRandom[:RandomLen-4], computed[:RandomLen-4]) != 1 {
			continue
		}

		timestamp := int64(binary.LittleEndian.Uint32(computed[RandomLen-4:]))
		createdAt := time.Unix(timestamp, 0)

		if tdiff := time.Since(createdAt).Abs(); tdiff > tolerateTimeSkewness {
			continue
		}

		return &ReadClientHelloResult{
			Hello:        hello,
			MatchedIndex: idx,
		}, nil
	}

	return nil, ErrBadDigest
}

func parseHandshake(r io.Reader) (*ClientHello, error) {
	//  A protocol version of "3,3" (meaning TLS 1.2) is given.
	header := [2]byte{}

	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("cannot read client version: %w", err)
	}

	hello := &ClientHello{}

	if _, err := io.ReadFull(r, hello.Random[:]); err != nil {
		return nil, fmt.Errorf("cannot read client random: %w", err)
	}

	if _, err := io.ReadFull(r, header[:1]); err != nil {
		return nil, fmt.Errorf("cannot read session ID length: %w", err)
	}

	hello.SessionID = make([]byte, int(header[0]))

	if _, err := io.ReadFull(r, hello.SessionID); err != nil {
		return nil, fmt.Errorf("cannot read session id: %w", err)
	}

	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("cannot read cipher suite length: %w", err)
	}

	cipherSuiteLen := int64(binary.BigEndian.Uint16(header[:]))

	// we do not care about picking up any cipher. we pick the first one,
	// so it is always should be present.
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("cannot read first cipher suite: %w", err)
	}

	hello.CipherSuite = binary.BigEndian.Uint16(header[:])

	if _, err := io.CopyN(io.Discard, r, cipherSuiteLen-2); err != nil {
		return nil, fmt.Errorf("cannot skip remaining cipher suites: %w", err)
	}

	if _, err := io.ReadFull(r, header[:1]); err != nil {
		return nil, fmt.Errorf("cannot read compression methods length: %w", err)
	}

	if _, err := io.CopyN(io.Discard, r, int64(header[0])); err != nil {
		return nil, fmt.Errorf("cannot skip compression methods: %w", err)
	}

	return hello, nil
}

func parseSNI(r io.Reader) ([]string, error) {
	header := [2]byte{}

	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("cannot read length of TLS extensions: %w", err)
	}

	extensionsLength := int64(binary.BigEndian.Uint16(header[:]))
	buf := &bytes.Buffer{}
	buf.Grow(int(extensionsLength))

	if _, err := io.CopyN(buf, r, extensionsLength); err != nil {
		return nil, fmt.Errorf("cannot read extensions: %w", err)
	}

	for buf.Len() > 0 {
		// 00 00 - assigned value for extension "server name"
		// 00 18 - 0x18 (24) bytes of "server name" extension data follows
		// 00 16 - 0x16 (22) bytes of first (and only) list entry follows
		// 00 - list entry is type 0x00 "DNS hostname"
		// 00 13 - 0x13 (19) bytes of hostname follows
		// 65 78 61 ... 6e 65 74 - "example.ulfheim.net"

		// 00 00 - assigned value for extension "server name"
		extTypeB := buf.Next(2)
		if len(extTypeB) != 2 {
			return nil, fmt.Errorf("cannot read extension type: %v", extTypeB)
		}

		// 00 18 - 0x18 (24) bytes of "server name" extension data follows
		lengthB := buf.Next(2)
		if len(lengthB) != 2 {
			return nil, fmt.Errorf("cannot read extension %v length: %v", extTypeB, lengthB)
		}
		length := int(binary.BigEndian.Uint16(lengthB))

		extDataB := buf.Next(length)
		if len(extDataB) != length {
			return nil, fmt.Errorf("cannot read extension %v data: len %d != %d", extTypeB, length, len(extDataB))
		}

		if !bytes.Equal(extTypeB, extTypeSNI[:]) {
			continue
		}

		buf.Reset()
		buf.Write(extDataB)

		// 00 16 - 0x16 (22) bytes of first (and only) list entry follows
		lengthB = buf.Next(2)
		if len(lengthB) != 2 {
			return nil, fmt.Errorf("cannot read the length of the SNI record: %v", lengthB)
		}

		length = int(binary.BigEndian.Uint16(lengthB))
		if length == 0 {
			return nil, nil
		}

		listType, err := buf.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("cannot read SNI list type: %w", err)
		}

		// 00 - list entry is type 0x00 "DNS hostname"
		if listType != sniDNSNamesListType {
			return nil, fmt.Errorf("incorrect SNI list type %#x", listType)
		}

		names := []string{}

		for buf.Len() > 0 {
			// 00 13 - 0x13 (19) bytes of hostname follows
			lengthB = buf.Next(2)
			if len(lengthB) != 2 {
				return nil, fmt.Errorf("incorrect length of the hostname: %v", lengthB)
			}
			length = int(binary.BigEndian.Uint16(lengthB))

			name := buf.Next(length)
			if len(name) != length {
				return nil, fmt.Errorf("incorrect length of SNI hostname: len %d != %d", length, len(name))
			}

			names = append(names, string(name))
		}

		return names, nil
	}

	return nil, nil
}
