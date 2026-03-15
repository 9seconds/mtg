package doppel

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
)

type ScoutConn struct {
	tls.Conn

	results *ScoutConnCollected
	rawBuf  *bytes.Buffer
}

func (s ScoutConn) Read(p []byte) (int, error) {
	buf := &bytes.Buffer{}

	for {
		if n, err := s.rawBuf.Read(p); err == nil {
			return n, nil
		}

		s.rawBuf.Reset()

		recordType, length, err := tls.ReadRecord(s.Conn, buf)
		if err != nil {
			return 0, err
		}

		s.results.Add(recordType)
		s.rawBuf.Write([]byte{recordType})
		s.rawBuf.Write(tls.TLSVersion[:])

		if err := binary.Write(s.rawBuf, binary.BigEndian, uint16(length)); err != nil {
			return 0, err
		}

		if _, err := io.Copy(s.rawBuf, buf); err != nil {
			return 0, err
		}
	}
}

func NewScoutConn(conn essentials.Conn, results *ScoutConnCollected) ScoutConn {
	rawBuf := &bytes.Buffer{}
	rawBuf.Grow(tls.MaxRecordSize)

	return ScoutConn{
		Conn:    tls.New(conn, false, false),
		results: results,
		rawBuf:  rawBuf,
	}
}
