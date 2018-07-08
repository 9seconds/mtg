package wrappers

import (
	"bytes"
	"crypto/aes"
	"encoding/binary"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net"

	"github.com/juju/errors"
)

const (
	mtprotoFrameMinMessageLength = 12
	mtprotoFrameMaxMessageLength = 16777216
)

var mtprotoFramePadding = []byte{0x04, 0x00, 0x00, 0x00}

type MTProtoFrame struct {
	conn       StreamReadWriteCloser
	readSeqNo  int32
	writeSeqNo int32
}

func (m *MTProtoFrame) Read() ([]byte, error) {
	buf := &bytes.Buffer{}
	sum := crc32.NewIEEE()
	writer := io.MultiWriter(buf, sum)

	for {
		buf.Reset()
		sum.Reset()
		if _, err := io.CopyN(writer, m.conn, 4); err != nil {
			return nil, errors.Annotate(err, "Cannot read frame padding")
		}
		if !bytes.Equal(buf.Bytes(), mtprotoFramePadding) {
			break
		}
	}

	messageLength := binary.LittleEndian.Uint32(buf.Bytes())
	m.LogDebug("Read MTProto frame",
		"messageLength", messageLength,
		"sequence_number", m.readSeqNo,
	)
	if messageLength%4 != 0 || messageLength < mtprotoFrameMinMessageLength || messageLength > mtprotoFrameMaxMessageLength {
		return nil, errors.Errorf("Incorrect frame message length %d", messageLength)
	}

	buf.Reset()
	buf.Grow(int(messageLength) - 4 - 4)
	if _, err := io.CopyN(writer, m.conn, int64(messageLength)-4-4); err != nil {
		return nil, errors.Annotate(err, "Cannot read the message frame")
	}

	var seqNo int32
	binary.Read(buf, binary.LittleEndian, &seqNo)
	if seqNo != m.readSeqNo {
		return nil, errors.Errorf("Unexpected sequence number %d (wait for %d)", seqNo, m.readSeqNo)
	}

	data, _ := ioutil.ReadAll(buf)
	buf.Reset()
	// write to buf, not to writer. This is because we are going to fetch
	// crc32 checksum.
	if _, err := io.CopyN(buf, m.conn, 4); err != nil {
		return nil, errors.Annotate(err, "Cannot read checksum")
	}

	checksum := binary.LittleEndian.Uint32(buf.Bytes())
	if checksum != sum.Sum32() {
		return nil, errors.Errorf("CRC32 checksum mismatch. Wait for %d, got %d", sum.Sum32(), checksum)
	}

	m.LogDebug("Read MTProto frame",
		"messageLength", messageLength,
		"sequence_number", m.readSeqNo,
		"dataLength", len(data),
		"checksum", checksum,
	)
	m.readSeqNo++

	return data, nil
}

func (m *MTProtoFrame) Write(p []byte) (int, error) {
	messageLength := 4 + 4 + len(p) + 4
	paddingLength := (aes.BlockSize - messageLength%aes.BlockSize) % aes.BlockSize

	buf := &bytes.Buffer{}
	buf.Grow(messageLength + paddingLength)

	binary.Write(buf, binary.LittleEndian, uint32(messageLength))
	binary.Write(buf, binary.LittleEndian, m.writeSeqNo)
	buf.Write(p)

	checksum := crc32.ChecksumIEEE(buf.Bytes())
	binary.Write(buf, binary.LittleEndian, checksum)
	buf.Write(bytes.Repeat(mtprotoFramePadding, paddingLength/4))

	m.LogDebug("Write MTProto frame",
		"length", len(p),
		"sequence_number", m.writeSeqNo,
		"crc32", checksum,
		"frame_length", buf.Len(),
	)
	m.writeSeqNo++

	_, err := m.conn.Write(buf.Bytes())

	return len(p), err
}

func (m *MTProtoFrame) LogDebug(msg string, data ...interface{}) {
	m.conn.LogDebug(msg, data...)
}

func (m *MTProtoFrame) LogInfo(msg string, data ...interface{}) {
	m.conn.LogInfo(msg, data...)
}

func (m *MTProtoFrame) LogWarn(msg string, data ...interface{}) {
	m.conn.LogWarn(msg, data...)
}

func (m *MTProtoFrame) LogError(msg string, data ...interface{}) {
	m.conn.LogError(msg, data...)
}

func (m *MTProtoFrame) LocalAddr() *net.TCPAddr {
	return m.conn.LocalAddr()
}

func (m *MTProtoFrame) RemoteAddr() *net.TCPAddr {
	return m.conn.RemoteAddr()
}

func (m *MTProtoFrame) Close() error {
	return m.conn.Close()
}

func NewMTProtoFrame(conn StreamReadWriteCloser, seqNo int32) PacketReadWriteCloser {
	return &MTProtoFrame{
		conn:       conn,
		readSeqNo:  seqNo,
		writeSeqNo: seqNo,
	}
}
