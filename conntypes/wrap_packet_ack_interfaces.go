package conntypes

import "io"

type PacketAckReader interface {
	Read(*ConnectionAcks) (Packet, error)
}

type PacketAckWriter interface {
	Write(Packet, *ConnectionAcks) error
}

type PacketAckCloser interface {
	io.Closer
}

type PacketAckReadCloser interface {
	PacketAckReader
	PacketAckCloser
}

type PacketAckWriteCloser interface {
	PacketAckWriter
	PacketAckCloser
}

type PacketAckReadWriter interface {
	PacketAckReader
	PacketAckWriter
}

type PacketAckReadWriteCloser interface {
	PacketAckReader
	PacketAckWriter
	PacketAckCloser
}

type PacketAckFullReadWriteCloser interface {
	Wrap
	PacketAckReadWriteCloser
}
