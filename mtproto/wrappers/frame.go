package wrappers

import (
	"bytes"
	"crypto/aes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/wrappers"
)

// Frame: { MessageLength(4) | SequenceNumber(4) | Message(???) | CRC32(4) [| padding(4), ...] }
const (
	frameRWCMinMessageLength = 12
	frameRWCMaxMessageLength = 16777216
)

var frameRWCPadding = []byte{0x04, 0x00, 0x00, 0x00}

type FrameRWC struct {
	wrappers.BufferedReader

	conn       wrappers.ReadWriteCloserWithAddr
	readSeqNo  int32
	writeSeqNo int32
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
	writeBuf.Write(bytes.Repeat(frameRWCPadding, paddingLength/4))

	_, err := f.conn.Write(writeBuf.Bytes())
	return len(buf), err
}

func (f *FrameRWC) Read(p []byte) (int, error) {
	return f.BufferedRead(p, func() error {
		buf := &bytes.Buffer{}
		sum := crc32.NewIEEE()
		writer := io.MultiWriter(buf, sum)

		for {
			buf.Reset()
			sum.Reset()
			if _, err := io.CopyN(writer, f.conn, 4); err != nil {
				return errors.Annotate(err, "Cannot read frame padding")
			}
			if !bytes.Equal(buf.Bytes(), frameRWCPadding[:]) {
				break
			}
		}

		messageLength := binary.LittleEndian.Uint32(buf.Bytes())
		if messageLength%4 != 0 || messageLength < frameRWCMinMessageLength || messageLength > frameRWCMaxMessageLength {
			return errors.Errorf("Incorrect frame message length %d", messageLength)
		}

		buf.Reset()
		buf.Grow(int(messageLength) - 4 - 4)
		if _, err := io.CopyN(writer, f.conn, int64(messageLength)-4-4); err != nil {
			return errors.Annotate(err, "Cannot read the message frame")
		}

		var seqNo int32
		binary.Read(buf, binary.LittleEndian, &seqNo)
		if seqNo != f.readSeqNo {
			return errors.Errorf("Unexpected sequence number %d (wait for %d)", seqNo, f.readSeqNo)
		}
		f.readSeqNo++

		data, _ := ioutil.ReadAll(buf)
		buf.Reset()
		// write to buf, not to writer. This is because we are going to fetch
		// crc32 checksum.
		if _, err := io.CopyN(buf, f.conn, 4); err != nil {
			return errors.Annotate(err, "Cannot read checksum")
		}
		checksum := binary.LittleEndian.Uint32(buf.Bytes())

		if checksum != sum.Sum32() {
			return errors.Errorf("CRC32 checksum mismatch. Wait for %d, got %d", sum.Sum32(), checksum)

		}
		f.Buffer.Write(data)

		return nil
	})
}

func (f *FrameRWC) Close() error {
	return f.conn.Close()
}

func (f *FrameRWC) LocalAddr() *net.TCPAddr {
	return f.conn.LocalAddr()
}

func (f *FrameRWC) RemoteAddr() *net.TCPAddr {
	return f.conn.RemoteAddr()
}

func NewFrameRWC(conn wrappers.ReadWriteCloserWithAddr, seqNo int32) wrappers.ReadWriteCloserWithAddr {
	return &FrameRWC{
		BufferedReader: wrappers.NewBufferedReader(),
		conn:           conn,
		readSeqNo:      seqNo,
		writeSeqNo:     seqNo,
	}
}
