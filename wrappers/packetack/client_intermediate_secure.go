package packetack

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"

	"github.com/9seconds/mtg/conntypes"
)

type wrapperClientIntermediateSecure struct {
	wrapperClientIntermediate
}

func (w *wrapperClientIntermediateSecure) Read(acks *conntypes.ConnectionAcks) (conntypes.Packet, error) {
	data, err := w.wrapperClientIntermediate.Read(acks)
	if err != nil {
		return nil, err
	}
	length := len(data) - (len(data) % 4)

	return data[:length], nil
}

func (w *wrapperClientIntermediateSecure) Write(packet conntypes.Packet, acks *conntypes.ConnectionAcks) error {
	if acks.Simple {
		if _, err := w.parent.Write(packet); err != nil {
			return fmt.Errorf("cannot send simpleacked packet: %w", err)
		}
		return nil
	}

	buf := bytes.Buffer{}
	paddingLength := rand.Intn(4)
	buf.Grow(4 + len(packet) + paddingLength)

	binary.Write(&buf, binary.LittleEndian, uint32(len(packet)+paddingLength))
	buf.Write(packet)
	buf.Write(make([]byte, paddingLength))

	if _, err := w.parent.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("cannot send packet: %w", err)
	}
	return nil
}

func NewClientIntermediateSecure(parent conntypes.StreamReadWriteCloser) conntypes.PacketAckFullReadWriteCloser {
	return &wrapperClientIntermediateSecure{
		wrapperClientIntermediate: wrapperClientIntermediate{
			parent: parent,
		},
	}
}
