package tls

import (
	"bufio"
	"bytes"

	"github.com/9seconds/mtg/v2/essentials"
)

const (
	SizeRecordType = 1
	SizeVersion    = 2
	SizeSize       = 2
	SizeHeader     = SizeRecordType + SizeVersion + SizeSize

	MaxRecordSize        = 16384
	MaxRecordPayloadSize = MaxRecordSize - SizeHeader
	DefaultBufferSize    = 4096

	TypeChangeCipherSpec = 0x14
	TypeHandshake        = 0x16
	TypeApplicationData  = 0x17
)

// TLS 1.2 is used for both TLS 1.2 and 1.3
var TLSVersion = [SizeVersion]byte{3, 3}

// Conn presents an established TLS 1.3 connection, after handshake
type Conn struct {
	essentials.Conn

	p *connPayload
}

type connPayload struct {
	readBuf      bytes.Buffer
	writeBuf     bytes.Buffer
	connBuffered *bufio.Reader
	read         bool
	write        bool
}

func (c Conn) Write(p []byte) (int, error) {
	if !c.p.write {
		return c.Conn.Write(p)
	}

	return len(p), WriteRecord(c.Conn, p)
}

func (c Conn) Read(p []byte) (int, error) {
	if !c.p.read {
		return c.Conn.Read(p)
	}

	for {
		if n, err := c.p.readBuf.Read(p); err == nil {
			return n, nil
		}

		recordType, _, err := ReadRecord(c.p.connBuffered, &c.p.readBuf)
		if err != nil {
			return 0, err
		}

		if recordType != TypeApplicationData {
			c.p.readBuf.Reset()
		}
	}
}

func New(conn essentials.Conn, read, write bool) Conn {
	newConn := Conn{
		Conn: conn,
		p: &connPayload{
			connBuffered: bufio.NewReaderSize(conn, DefaultBufferSize),
			read:         read,
			write:        write,
		},
	}

	newConn.p.readBuf.Grow(DefaultBufferSize)
	newConn.p.writeBuf.Grow(DefaultBufferSize)

	return newConn
}
