package doppel

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/dolonet/mtg-multi/essentials"
	"github.com/dolonet/mtg-multi/mtglib/internal/tls"
)

type ScoutConn struct {
	tls.Conn

	results *ScoutConnCollected
	rawBuf  *bytes.Buffer
	seenCCS bool
}

func (s *ScoutConn) Read(p []byte) (int, error) {
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

		if recordType == tls.TypeChangeCipherSpec {
			s.seenCCS = true
		}

		s.results.Add(recordType, int(length))
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

func (s *ScoutConn) Write(p []byte) (int, error) {
	if s.seenCCS {
		s.results.MarkWrite()
	}

	return s.Conn.Write(p)
}

func NewScoutConn(conn essentials.Conn, results *ScoutConnCollected) *ScoutConn {
	rawBuf := &bytes.Buffer{}
	rawBuf.Grow(tls.MaxRecordSize)

	return &ScoutConn{
		Conn:    tls.New(conn, false, false),
		results: results,
		rawBuf:  rawBuf,
	}
}
