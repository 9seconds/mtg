package wrappers

import (
	"bytes"
	"crypto/aes"
	"encoding/binary"
	"hash/crc32"
	"io"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/wrappers"
)

// Frame: { MessageLength(4) | SequenceNumber(4) | Message(???) | CRC32(4) [| padding(4), ...] }
const (
	frameRWCMinMessageLength = 12
	frameRWCMaxMessageLength = 16777216
)

var frameRWCPadding = [4]byte{0x04, 0x00, 0x00, 0x00}

type FrameRWC struct {
	conn wrappers.ReadWriteCloserWithAddr

	readSeqNo  int32
	writeSeqNo int32
	readBuf    *bytes.Buffer
}

func (f *FrameRWC) Write(buf []byte) (int, error) {
	writeBuf := &bytes.Buffer{}

	// 4 - len bytes
	// 4 - seq bytes
	// . - message
	// 4 - crc32
	messageLength := 4 + 4 + len(buf) + 4
	paddingLength := (aes.BlockSize - messageLength%aes.BlockSize) % aes.BlockSize
	writeBuf.Grow(messageLength + paddingLength)

	binary.Write(writeBuf, binary.LittleEndian, uint32(messageLength))
	binary.Write(writeBuf, binary.LittleEndian, f.writeSeqNo)
	writeBuf.Write(buf)
	f.writeSeqNo++

	checksum := crc32.ChecksumIEEE(writeBuf.Bytes())
	binary.Write(writeBuf, binary.LittleEndian, checksum)
	writeBuf.Write(bytes.Repeat(frameRWCPadding[:], paddingLength/4))

	_, err := f.conn.Write(writeBuf.Bytes())
	return len(buf), err
}

func (f *FrameRWC) Read(p []byte) (int, error) {
	if f.readBuf.Len() > 0 {
		return f.flush(p)
	}

	buf := &bytes.Buffer{}
	for {
		buf.Reset()
		if _, err := io.CopyN(buf, f.conn, 4); err != nil {
			return 0, errors.Annotate(err, "Cannot read frame padding")
		}
		if !bytes.Equal(buf.Bytes(), frameRWCPadding[:]) {
			break
		}
	}

	messageLength := binary.LittleEndian.Uint32(buf.Bytes())
	if messageLength%4 != 0 || messageLength < frameRWCMinMessageLength || messageLength > frameRWCMaxMessageLength {
		return 0, errors.Errorf("Incorrect frame message length %d", messageLength)
	}
	sum := crc32.NewIEEE()
	sum.Write(buf.Bytes())

	buf.Reset()
	buf.Grow(int(messageLength) - 4) // -4 because we already read the first number
	if _, err := io.CopyN(buf, f.conn, int64(messageLength)-4); err != nil {
		return 0, errors.Annotate(err, "Cannot read the message frame")
	}
	sum.Write(buf.Bytes())

	var seqNo int32
	binary.Read(buf, binary.LittleEndian, seqNo)
	if seqNo != f.readSeqNo {
		return 0, errors.Errorf("Unexpected sequence number %d (wait for %d)", seqNo, f.readSeqNo)
	}
	f.readSeqNo++

	data := buf.Bytes()[:int(messageLength)-4-4-4]
	checksum := binary.LittleEndian.Uint32(buf.Bytes()[int(messageLength)-4-4-4:])
	if checksum != sum.Sum32() {
		return 0, errors.Errorf("CRC32 checksum mismatch. Wait for %d, got %d", sum.Sum32(), checksum)

	}
	f.readBuf.Write(data)

	return f.flush(p)
}

func (f *FrameRWC) Close() error {
	return f.conn.Close()
}

func (f *FrameRWC) Addr() *net.TCPAddr {
	return f.conn.Addr()
}

func (f *FrameRWC) flush(p []byte) (int, error) {
	sizeToRead := len(p)
	if f.readBuf.Len() < sizeToRead {
		sizeToRead = f.readBuf.Len()
	}

	data := f.readBuf.Bytes()
	copy(p, data[:sizeToRead])
	if sizeToRead == f.readBuf.Len() {
		f.readBuf.Reset()
	} else {
		f.readBuf = bytes.NewBuffer(data[sizeToRead:])
	}

	return sizeToRead, nil
}

func NewFrameRWC(conn wrappers.ReadWriteCloserWithAddr, seqNo int32) wrappers.ReadWriteCloserWithAddr {
	return &FrameRWC{
		conn:       conn,
		readSeqNo:  seqNo,
		writeSeqNo: seqNo,
		readBuf:    &bytes.Buffer{},
	}
}
