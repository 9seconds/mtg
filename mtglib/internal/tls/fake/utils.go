package fake

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/dolonet/mtg-multi/mtglib/internal/tls"
)

const (
	maxFragmentsCount = 10
)

var ErrTooManyFragments = errors.New("too many fragments")

// https://datatracker.ietf.org/doc/html/rfc5246#section-6.2.1
// client hello can be fragmented in a series of packets:
//
//	Bytes on the wire:
//
// 16 03 01 00 F8 01 00 00 F4 03 03 [32 bytes random] [session_id] [ciphers] [SNI...]
// ├─────────────┤├──────────────────────────────────────────────────────────────────┤
//
//	TLS record       Payload (248 bytes)
//	header (5B)
//
//	16    = Handshake
//	03 01 = TLS 1.0 (record layer version)
//	00 F8 = 248 bytes follow
//
//	01       = ClientHello (handshake type)
//	00 00 F4 = 244 bytes of handshake body
//	03 03    = TLS 1.2 (actual protocol version)
//	...rest of ClientHello...
//
// Fragmented record look like:
//
//	Record 1:
//
// 16 03 01 00 03 01 00 00
// ├─────────────┤├──────┤
//
//	TLS header    3 bytes of payload
//
//	16    = Handshake
//	03 01 = TLS 1.0
//	00 03 = only 3 bytes follow
//
//	01       = ClientHello type
//	00 00    = first 2 bytes of the uint24 length (INCOMPLETE!)
//
// Record 2:
// 16 03 01 00 F5 F4 03 03 [32 bytes random] [session_id] [ciphers] [SNI...]
// ├─────────────┤├────────────────────────────────────────────────────────────┤
//
//	TLS header    remaining 245 bytes of payload
//
//	16    = Handshake
//	03 01 = TLS 1.0
//	00 F5 = 245 bytes follow
//
//	F4    = last byte of uint24 length (now complete: 00 00 F4 = 244)
//	03 03 = TLS 1.2
//	...rest of ClientHello continues...
//
// So it means that there could be a series of handshake packets of different
// lengths. The goal of this function is to concatenate these fragments.
type fragmentedHandshakeReader struct {
	r             io.Reader
	buf           bytes.Buffer
	readFragments int
}

func (f *fragmentedHandshakeReader) Read(p []byte) (int, error) {
	if n, err := f.buf.Read(p); err == nil {
		return n, nil
	}

	f.buf.Reset()

	for f.buf.Len() == 0 {
		if f.readFragments > maxFragmentsCount {
			return 0, ErrTooManyFragments
		}

		if err := f.parseNextFragment(); err != nil {
			return 0, err
		}

		f.readFragments++
	}

	return f.buf.Read(p)
}

func (f *fragmentedHandshakeReader) parseNextFragment() error {
	// record_type(1) + version(2) + size(2)
	//   16 - type is 0x16 (handshake record)
	//   03 01 - protocol version is "3,1" (also known as TLS 1.0)
	//   00 f8 - 0xF8 (248) bytes of handshake message follows
	header := [1 + 2 + 2]byte{}

	if _, err := io.ReadFull(f.r, header[:]); err != nil {
		return fmt.Errorf("cannot read record header: %w", err)
	}

	if header[0] != tls.TypeHandshake {
		return fmt.Errorf("unexpected record type %#x", header[0])
	}

	if header[1] != 3 || header[2] != 1 {
		return fmt.Errorf("unexpected protocol version %#x %#x", header[1], header[2])
	}

	length := int64(binary.BigEndian.Uint16(header[3:]))
	_, err := io.CopyN(&f.buf, f.r, length)

	return err
}

func parseClientHello(r io.Reader) (*bytes.Buffer, *bytes.Buffer, error) {
	r = &fragmentedHandshakeReader{r: r}
	header := [1 + 3]byte{}

	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, nil, fmt.Errorf("cannot read handshake header: %w", err)
	}

	if header[0] != TypeHandshakeClient {
		return nil, nil, fmt.Errorf("incorrect handshake type: %#x", header[0])
	}

	// unfortunately there is not uint24 in golang, so we just reuse header
	header[0] = 0
	length := int64(binary.BigEndian.Uint32(header[:]))

	clientHelloCopy := &bytes.Buffer{}
	clientHelloCopy.Write([]byte{tls.TypeHandshake, 3, 1})
	binary.Write( //nolint: errcheck
		clientHelloCopy,
		binary.BigEndian,
		// 1 for handshake type
		// 3 for handshake length
		uint16(1+3+length),
	)
	clientHelloCopy.WriteByte(TypeHandshakeClient)
	clientHelloCopy.Write(header[1:])

	handshakeCopy := &bytes.Buffer{}
	writer := io.MultiWriter(clientHelloCopy, handshakeCopy)

	_, err := io.CopyN(writer, r, length)

	return clientHelloCopy, handshakeCopy, err
}
