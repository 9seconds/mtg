package fake

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
)

const (
	TypeHandshakeClient = 0x01

	RandomLen = 32
	// record_type(1) + version(2) + size(2) + handshake_type(1) + uint24_length(3) + client_version(2)
	RandomOffset = 1 + 2 + 2 + 1 + 3 + 2

	sniDNSNamesListType = 0

	// maxContinuationRecords limits the number of continuation TLS records
	// that reassembleTLSHandshake will read. This prevents resource exhaustion
	// from adversarial fragmentation.
	maxContinuationRecords = 10
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

func ReadClientHello(
	conn net.Conn,
	secret []byte,
	hostname string,
	tolerateTimeSkewness time.Duration,
) (*ClientHello, error) {
	if err := conn.SetReadDeadline(time.Now().Add(ClientHelloReadTimeout)); err != nil {
		return nil, fmt.Errorf("cannot set read deadline: %w", err)
	}
	defer conn.SetReadDeadline(resetDeadline) //nolint: errcheck

	// This is how FakeTLS is organized:
	//  1. We create sha256 HMAC with a given secret
	//  2. We dump there a whole TLS frame except of the fact that random
	//     is filled with all zeroes
	//  3. Digest is computed. This digest should be XORed with
	//     original client random
	//  4. New digest should be all 0 except of last 4 bytes
	//  5. Last 4 bytes are little endian uint32 of UNIX timestamp when
	//     this message was created.
	reassembled, err := reassembleTLSHandshake(conn)
	if err != nil {
		return nil, fmt.Errorf("cannot reassemble TLS records: %w", err)
	}

	handshakeCopyBuf := &bytes.Buffer{}
	reader := io.TeeReader(reassembled, handshakeCopyBuf)

	// Skip the TLS record header (validated during reassembly).
	// The header still flows through TeeReader into handshakeCopyBuf for HMAC.
	if _, err = io.CopyN(io.Discard, reader, tls.SizeHeader); err != nil {
		return nil, fmt.Errorf("cannot skip tls header: %w", err)
	}

	reader, err = parseHandshakeHeader(reader)
	if err != nil {
		return nil, fmt.Errorf("cannot parse handshake header: %w", err)
	}

	hello, err := parseHandshake(reader)
	if err != nil {
		return nil, fmt.Errorf("cannot parse handshake: %w", err)
	}

	sniHostnames, err := parseSNI(reader)
	if err != nil {
		return nil, fmt.Errorf("cannot parse SNI: %w", err)
	}

	if !slices.Contains(sniHostnames, hostname) {
		return nil, fmt.Errorf("cannot find %s in %v", hostname, sniHostnames)
	}

	digest := hmac.New(sha256.New, secret)
	// we write a copy of the handshake with client random all nullified.
	digest.Write(handshakeCopyBuf.Next(RandomOffset))
	handshakeCopyBuf.Next(RandomLen)
	digest.Write(emptyRandom[:])
	digest.Write(handshakeCopyBuf.Bytes())

	computed := digest.Sum(nil)

	for i := range RandomLen {
		computed[i] ^= hello.Random[i]
	}

	if subtle.ConstantTimeCompare(emptyRandom[:RandomLen-4], computed[:RandomLen-4]) != 1 {
		return nil, ErrBadDigest
	}

	timestamp := int64(binary.LittleEndian.Uint32(computed[RandomLen-4:]))
	createdAt := time.Unix(timestamp, 0)

	if tdiff := time.Since(createdAt).Abs(); tdiff > tolerateTimeSkewness {
		return nil, fmt.Errorf("timestamp %q is too old %s", createdAt, tdiff)
	}

	return hello, nil
}

// reassembleTLSHandshake reads one or more TLS records from conn,
// validates the record type and version, and reassembles fragmented
// handshake payloads into a single TLS record.
//
// Per RFC 5246 Section 6.2.1, handshake messages may be fragmented
// across multiple TLS records. DPI bypass tools like ByeDPI use this
// to evade censorship.
//
// The returned buffer contains the full TLS record (header + payload)
// so that callers can include the header in HMAC computation.
func reassembleTLSHandshake(conn io.Reader) (*bytes.Buffer, error) {
	header := [tls.SizeHeader]byte{}

	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return nil, fmt.Errorf("cannot read record header: %w", err)
	}

	length := int64(binary.BigEndian.Uint16(header[3:]))
	payload := &bytes.Buffer{}

	if _, err := io.CopyN(payload, conn, length); err != nil {
		return nil, fmt.Errorf("cannot read record payload: %w", err)
	}

	if header[0] != tls.TypeHandshake {
		return nil, fmt.Errorf("unexpected record type %#x", header[0])
	}

	if header[1] != 3 || header[2] != 1 {
		return nil, fmt.Errorf("unexpected protocol version %#x %#x", header[1], header[2])
	}

	// Reassemble fragmented payload. continuationCount caps the total
	// number of continuation records across both phases below.
	continuationCount := 0

	// Phase 1: read continuation records until we have at least the
	// 4-byte handshake header (type + uint24 length) to determine the
	// expected total size.
	for ; payload.Len() < 4 && continuationCount < maxContinuationRecords; continuationCount++ {
		prevLen := payload.Len()

		if err := readContinuationRecord(conn, payload); err != nil {
			payload.Truncate(prevLen) // discard partial data on error

			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break // no more records — let downstream parsing handle what we have
			}

			return nil, err
		}
	}

	// Phase 2: we know the expected handshake size — read remaining
	// continuation records until the payload is complete.
	if payload.Len() >= 4 {
		p := payload.Bytes()
		expectedTotal := 4 + (int(p[1])<<16 | int(p[2])<<8 | int(p[3]))

		if expectedTotal > 0xFFFF {
			return nil, fmt.Errorf("handshake message too large: %d bytes", expectedTotal)
		}

		for ; payload.Len() < expectedTotal && continuationCount < maxContinuationRecords; continuationCount++ {
			if err := readContinuationRecord(conn, payload); err != nil {
				return nil, err
			}
		}

		if payload.Len() < expectedTotal {
			return nil, fmt.Errorf("cannot reassemble handshake: too many continuation records")
		}

		payload.Truncate(expectedTotal)
	}

	if payload.Len() > 0xFFFF {
		return nil, fmt.Errorf("reassembled payload too large: %d bytes", payload.Len())
	}

	// Reconstruct a single TLS record with the reassembled payload.
	result := &bytes.Buffer{}
	result.Grow(tls.SizeHeader + payload.Len())
	result.Write(header[:3])
	binary.Write(result, binary.BigEndian, uint16(payload.Len())) //nolint:errcheck // bytes.Buffer.Write never fails
	result.Write(payload.Bytes())

	return result, nil
}

// readContinuationRecord reads the next TLS record header and appends its
// full payload to dst. It returns an error if the record is not a handshake
// record.
func readContinuationRecord(conn io.Reader, dst *bytes.Buffer) error {
	nextHeader := [tls.SizeHeader]byte{}

	if _, err := io.ReadFull(conn, nextHeader[:]); err != nil {
		return fmt.Errorf("cannot read continuation record header: %w", err)
	}

	if nextHeader[0] != tls.TypeHandshake {
		return fmt.Errorf("unexpected continuation record type %#x", nextHeader[0])
	}

	if nextHeader[1] != 3 || nextHeader[2] != 1 {
		return fmt.Errorf("unexpected continuation record version %#x %#x", nextHeader[1], nextHeader[2])
	}

	nextLength := int64(binary.BigEndian.Uint16(nextHeader[3:]))

	if nextLength == 0 {
		return fmt.Errorf("zero-length continuation record")
	}

	if _, err := io.CopyN(dst, conn, nextLength); err != nil {
		return fmt.Errorf("cannot read continuation record payload: %w", err)
	}

	return nil
}

func parseHandshakeHeader(r io.Reader) (io.Reader, error) {
	// type(1) + size(3 / uint24)
	// 01 - handshake message type 0x01 (client hello)
	// 00 00 f4 - 0xF4 (244) bytes of client hello data follows
	header := [1 + 3]byte{}

	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("cannot read handshake header: %w", err)
	}

	if header[0] != TypeHandshakeClient {
		return nil, fmt.Errorf("incorrect handshake type: %#x", header[0])
	}

	// unfortunately there is not uint24 in golang, so we just reust header
	header[0] = 0

	length := int64(binary.BigEndian.Uint32(header[:]))
	buf := &bytes.Buffer{}

	_, err := io.CopyN(buf, r, length)

	return buf, err
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
