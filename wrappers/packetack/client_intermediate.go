package packetack

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
)

const clientIntermediateQuickAckLength = 0x80000000

type wrapperClientIntermediate struct {
	parent conntypes.StreamReadWriteCloser
}

func (w *wrapperClientIntermediate) Read(acks *conntypes.ConnectionAcks) (conntypes.Packet, error) {
	buf := bytes.Buffer{}

	buf.Grow(4)
	if _, err := io.CopyN(&buf, w.parent, 4); err != nil {
		return nil, fmt.Errorf("cannot read message length: %w", err)
	}
	length := binary.LittleEndian.Uint32(buf.Bytes())

	if length > clientIntermediateQuickAckLength {
		acks.Quick = true
		length -= clientIntermediateQuickAckLength
	}

	buf.Reset()
	buf.Grow(int(length))
	if _, err := io.CopyN(&buf, w.parent, int64(length)); err != nil {
		return nil, fmt.Errorf("cannot read the message: %w", err)
	}

	return buf.Bytes(), nil
}

func (w *wrapperClientIntermediate) Write(packet conntypes.Packet, acks *conntypes.ConnectionAcks) error {
	if acks.Simple {
		if _, err := w.parent.Write(packet); err != nil {
			return fmt.Errorf("cannot send simpleacked packet: %w", err)
		}
		return nil
	}

	length := [4]byte{}
	binary.LittleEndian.PutUint32(length[:], uint32(len(packet)))

	if _, err := w.parent.Write(append(length[:], packet...)); err != nil {
		return fmt.Errorf("cannot send packet: %w", err)
	}
	return nil
}

func (w *wrapperClientIntermediate) Close() error {
	return w.parent.Close()
}

func (w *wrapperClientIntermediate) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperClientIntermediate) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperClientIntermediate) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func (w *wrapperClientIntermediate) Logger() *zap.SugaredLogger {
	return w.parent.Logger().Named("client-intermediate")
}

func NewClientIntermediate(parent conntypes.StreamReadWriteCloser) conntypes.PacketAckFullReadWriteCloser {
	return &wrapperClientIntermediate{
		parent: parent,
	}
}
