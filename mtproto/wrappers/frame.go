package wrappers

import (
	"bytes"
	"crypto/aes"
	"encoding/binary"
	"hash/crc32"
	"io"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
)

// Frame: { MessageLength(4) | SequenceNumber(4) | Message(???) | CRC32(4) [| padding(4), ...] }
const (
	frameRWCMinMessageLength = 12
	frameRWCMaxMessageLength = 16777216
)

var frameRWCPadding = [4]byte{0x04, 0x00, 0x00, 0x00}

type FrameRWC struct {
	conn mtproto.BytesRWC

	readSeqNo  int32
	writeSeqNo int32
	readBuf    *bytes.Buffer
}

func (f *FrameRWC) Write(buf *bytes.Buffer) (int, error) {
	writeBuf := mtproto.GetBuffer()
	defer mtproto.ReturnBuffer(writeBuf)

	// 4 - len bytes
	// 4 - seq bytes
	// . - message
	// 4 - crc32
	messageLength := 4 + 4 + buf.Len() + 4
	paddingLength := (aes.BlockSize - messageLength%aes.BlockSize) % aes.BlockSize
	writeBuf.Grow(messageLength + paddingLength)

	binary.Write(writeBuf, binary.LittleEndian, uint32(messageLength))
	binary.Write(writeBuf, binary.LittleEndian, f.writeSeqNo)
	writeBuf.Write(buf.Bytes())
	f.writeSeqNo++

	checksum := crc32.ChecksumIEEE(writeBuf.Bytes())
	binary.Write(writeBuf, binary.LittleEndian, checksum)
	writeBuf.Write(bytes.Repeat(frameRWCPadding[:], paddingLength/4))

	return f.conn.Write(writeBuf)
}

func (f *FrameRWC) Read(p []byte) (int, error) {
	if f.readBuf.Len() > 0 {
		return f.flush(p)
	}

	buf := mtproto.GetBuffer()
	defer mtproto.ReturnBuffer(buf)

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
	defer mtproto.ReturnBuffer(f.readBuf)
	return f.conn.Close()
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
		newBuf := mtproto.GetBuffer()
		newBuf.Write(data[sizeToRead:])

		mtproto.ReturnBuffer(f.readBuf)
		f.readBuf = newBuf
	}

	return sizeToRead, nil
}

func NewFrameRWC(conn mtproto.BytesRWC, seqNo int32) mtproto.BytesRWC {
	return &FrameRWC{
		conn:       conn,
		readSeqNo:  seqNo,
		writeSeqNo: seqNo,
		readBuf:    mtproto.GetBuffer(),
	}
}
