package tlstypes

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const recordMaxChunkSize = 16384 + 24

type Record struct {
	Type    RecordType
	Version Version
	Data    Byter
}

func (r Record) Bytes() []byte {
	buf := bytes.Buffer{}
	data := r.Data.Bytes()

	buf.WriteByte(byte(r.Type))
	buf.Write(r.Version.Bytes())
	binary.Write(&buf, binary.BigEndian, uint16(len(data))) // nolint: errcheck
	buf.Write(data)

	return buf.Bytes()
}

func ReadRecord(reader io.Reader) (Record, error) {
	buf := [2]byte{}
	rec := Record{}

	if _, err := io.ReadFull(reader, buf[:1]); err != nil {
		return rec, fmt.Errorf("cannot read record type: %w", err)
	}

	rec.Type = RecordType(buf[0])

	if _, err := io.ReadFull(reader, buf[:]); err != nil {
		return rec, fmt.Errorf("cannot read version: %w", err)
	}

	switch {
	case bytes.Equal(buf[:], Version13Bytes):
		rec.Version = Version13
	case bytes.Equal(buf[:], Version12Bytes):
		rec.Version = Version12
	case bytes.Equal(buf[:], Version11Bytes):
		rec.Version = Version11
	case bytes.Equal(buf[:], Version10Bytes):
		rec.Version = Version10
	}

	if _, err := io.ReadFull(reader, buf[:]); err != nil {
		return rec, fmt.Errorf("cannot read data length: %w", err)
	}

	data := make([]byte, binary.BigEndian.Uint16(buf[:]))
	if _, err := io.ReadFull(reader, data); err != nil {
		return rec, fmt.Errorf("cannot read data: %w", err)
	}

	rec.Data = RawBytes(data)

	return rec, nil
}

func MakeRecords(raw []byte) (arr []Record) {
	for len(raw) > 0 {
		chunkSize := recordMaxChunkSize
		if chunkSize > len(raw) {
			chunkSize = len(raw)
		}

		arr = append(arr, Record{
			Type:    RecordTypeApplicationData,
			Version: Version12,
			Data:    RawBytes(raw[:chunkSize]),
		})
		raw = raw[chunkSize:]
	}

	return
}
