package tls

import (
	"encoding/binary"

	"github.com/stretchr/testify/mock"
)

type WriterMock struct {
	mock.Mock
}

func (m *WriterMock) Write(p []byte) (int, error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

// makeTLSRecord builds a raw TLS record from hardcoded offsets:
// type(1) + version(2, {3,3}) + length(2, big-endian) + payload.
func MakeTLSRecord(recordType byte, payload []byte) []byte {
	buf := make([]byte, 5+len(payload))

	buf[0] = recordType
	buf[1] = 3
	buf[2] = 3
	binary.BigEndian.PutUint16(buf[3:5], uint16(len(payload)))
	copy(buf[5:], payload)

	return buf
}
