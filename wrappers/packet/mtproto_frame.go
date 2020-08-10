package packet

import (
	"bytes"
	"crypto/aes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net"

	"github.com/9seconds/mtg/conntypes"
	"go.uber.org/zap"
)

const (
	mtprotoFrameMinMessageLength = 12
	mtprotoFrameMaxMessageLength = 16777216
)

var mtprotoFramePadding = []byte{0x04, 0x00, 0x00, 0x00}

// MTProtoFrame is a wrapper which converts written data to the MTProtoFrame.
// The format of the frame:
//
// [ MSGLEN(4) | SEQNO(4) | MSG(...) | CRC32(4) | PADDING(4*x) ]
//
// MSGLEN is the length of the message + len of seqno and msglen.
// SEQNO is the number of frame in the receive/send sequence. If client
//   sends a message with SeqNo 18, it has to receive message with SeqNo 18.
// MSG is the data which has to be written
// CRC32 is the CRC32 checksum of MSGLEN + SEQNO + MSG
// PADDING is custom padding schema to complete frame length to such that
//    len(frame) % 16 == 0
type wrapperMtprotoFrame struct {
	parent     conntypes.StreamReadWriteCloser
	logger     *zap.SugaredLogger
	readSeqNo  int32
	writeSeqNo int32
}

func (w *wrapperMtprotoFrame) Read() (conntypes.Packet, error) { // nolint: funlen
	buf := &bytes.Buffer{}

	sum := crc32.NewIEEE()
	writer := io.MultiWriter(buf, sum)

	for {
		buf.Reset()
		sum.Reset()

		if _, err := io.CopyN(writer, w.parent, 4); err != nil {
			return nil, fmt.Errorf("cannot read frame padding: %w", err)
		}

		if !bytes.Equal(buf.Bytes(), mtprotoFramePadding) {
			break
		}
	}

	messageLength := binary.LittleEndian.Uint32(buf.Bytes())
	w.logger.Debugw("Read MTProto frame",
		"messageLength", messageLength,
		"sequence_number", w.readSeqNo,
	)

	if messageLength%4 != 0 || messageLength < mtprotoFrameMinMessageLength ||
		messageLength > mtprotoFrameMaxMessageLength {
		return nil, fmt.Errorf("incorrect frame message length %d", messageLength)
	}

	buf.Reset()

	if _, err := io.CopyN(writer, w.parent, int64(messageLength)-4-4); err != nil {
		return nil, fmt.Errorf("cannot read the message frame: %w", err)
	}

	var seqNo int32

	binary.Read(buf, binary.LittleEndian, &seqNo) // nolint: errcheck

	if seqNo != w.readSeqNo {
		return nil, fmt.Errorf("unexpected sequence number %d (wait for %d)", seqNo, w.readSeqNo)
	}

	data, _ := ioutil.ReadAll(buf)
	buf.Reset()
	// write to buf, not to writer. This is because we are going to fetch
	// crc32 checksum.
	if _, err := io.CopyN(buf, w.parent, 4); err != nil {
		return nil, fmt.Errorf("cannot read checksum: %w", err)
	}

	checksum := binary.LittleEndian.Uint32(buf.Bytes())
	if checksum != sum.Sum32() {
		return nil, fmt.Errorf("CRC32 checksum mismatch. wait for %d, got %d", sum.Sum32(), checksum)
	}

	w.logger.Debugw("Read MTProto frame",
		"messageLength", messageLength,
		"sequence_number", w.readSeqNo,
		"dataLength", len(data),
		"checksum", checksum,
	)
	w.readSeqNo++

	return data, nil
}

func (w *wrapperMtprotoFrame) Write(p conntypes.Packet) error {
	messageLength := 4 + 4 + len(p) + 4
	paddingLength := (aes.BlockSize - messageLength%aes.BlockSize) % aes.BlockSize

	buf := &bytes.Buffer{}

	binary.Write(buf, binary.LittleEndian, uint32(messageLength)) // nolint: errcheck
	binary.Write(buf, binary.LittleEndian, w.writeSeqNo)          // nolint: errcheck
	buf.Write(p)

	checksum := crc32.ChecksumIEEE(buf.Bytes())
	binary.Write(buf, binary.LittleEndian, checksum) // nolint: errcheck
	buf.Write(bytes.Repeat(mtprotoFramePadding, paddingLength/4))

	w.logger.Debugw("Write MTProto frame",
		"length", len(p),
		"sequence_number", w.writeSeqNo,
		"crc32", checksum,
		"frame_length", buf.Len(),
	)
	w.writeSeqNo++

	_, err := w.parent.Write(buf.Bytes())

	return err
}

func (w *wrapperMtprotoFrame) Close() error {
	return w.parent.Close()
}

func (w *wrapperMtprotoFrame) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperMtprotoFrame) Logger() *zap.SugaredLogger {
	return w.logger
}

func (w *wrapperMtprotoFrame) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperMtprotoFrame) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func NewMtprotoFrame(parent conntypes.StreamReadWriteCloser, seqNo int32) conntypes.PacketReadWriteCloser {
	return &wrapperMtprotoFrame{
		parent:     parent,
		logger:     parent.Logger().Named("mtproto-frame"),
		readSeqNo:  seqNo,
		writeSeqNo: seqNo,
	}
}
