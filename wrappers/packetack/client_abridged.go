package packetack

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/utils"
)

const (
	clientAbridgedSmallPacketLength = 0x7f
	clientAbridgedQuickAckLength    = 0x80
	clientAbridgedLargePacketLength = 16777216 // 256 ^ 3
)

type wrapperClientAbridged struct {
	parent conntypes.StreamReadWriteCloser
}

func (w *wrapperClientAbridged) Read(acks *conntypes.ConnectionAcks) (conntypes.Packet, error) {
	buf := bytes.Buffer{}

	buf.Grow(1)

	if _, err := io.CopyN(&buf, w.parent, 1); err != nil {
		return nil, fmt.Errorf("cannot read message length: %w", err)
	}

	msgLength := uint32(buf.Bytes()[0])
	buf.Reset()

	if msgLength >= clientAbridgedQuickAckLength {
		acks.Quick = true
		msgLength -= clientAbridgedQuickAckLength
	}

	if msgLength == clientAbridgedSmallPacketLength {
		buf.Grow(3)

		if _, err := io.CopyN(&buf, w.parent, 3); err != nil {
			return nil, fmt.Errorf("cannot read correct message length: %w", err)
		}

		number := utils.Uint24{}
		copy(number[:], buf.Bytes())
		msgLength = utils.FromUint24(number)
	}

	msgLength *= 4

	buf.Reset()
	buf.Grow(int(msgLength))

	if _, err := io.CopyN(&buf, w.parent, int64(msgLength)); err != nil {
		return nil, fmt.Errorf("cannot read message: %w", err)
	}

	return conntypes.Packet(buf.Bytes()), nil
}

func (w *wrapperClientAbridged) Write(packet conntypes.Packet, acks *conntypes.ConnectionAcks) error {
	if len(packet)%4 != 0 {
		return fmt.Errorf("incorrect packet length %d", len(packet))
	}

	if acks.Simple {
		if _, err := w.parent.Write(utils.ReverseBytes(packet)); err != nil {
			return fmt.Errorf("cannot send a simpleacked packet: %w", err)
		}

		return nil
	}

	packetLength := len(packet) / 4

	switch {
	case packetLength < clientAbridgedSmallPacketLength:
		data := append([]byte{byte(packetLength)}, packet...)
		if _, err := w.parent.Write(data); err != nil {
			return fmt.Errorf("cannot send small packet: %w", err)
		}

		return nil
	case packetLength < clientAbridgedLargePacketLength:
		length24 := utils.ToUint24(uint32(packetLength))

		buf := acquireClientBytesBuffer()
		defer releaseClientBytesBuffer(buf)

		buf.WriteByte(byte(clientAbridgedSmallPacketLength))
		buf.Write(length24[:])
		buf.Write(packet)

		if _, err := w.parent.Write(buf.Bytes()); err != nil {
			return fmt.Errorf("cannot send large packet: %w", err)
		}

		return nil
	}

	return fmt.Errorf("packet is too big: %d", len(packet))
}

func (w *wrapperClientAbridged) Close() error {
	return w.parent.Close()
}

func (w *wrapperClientAbridged) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperClientAbridged) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperClientAbridged) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func (w *wrapperClientAbridged) Logger() *zap.SugaredLogger {
	return w.parent.Logger().Named("client-abridged")
}

func NewClientAbridged(parent conntypes.StreamReadWriteCloser) conntypes.PacketAckFullReadWriteCloser {
	return &wrapperClientAbridged{
		parent: parent,
	}
}
