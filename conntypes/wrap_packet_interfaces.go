package conntypes

import "io"

type BasePacketReader interface {
	Read() (Packet, error)
}

type BasePacketWriter interface {
	Write(Packet) error
}

type PacketReader interface {
	Wrap
	BasePacketReader
}

type PacketWriter interface {
	Wrap
	BasePacketWriter
}

type PacketCloser interface {
	Wrap
	io.Closer
}

type PacketReadCloser interface {
	Wrap
	BasePacketReader
	io.Closer
}

type PacketWriteCloser interface {
	Wrap
	BasePacketWriter
	io.Closer
}

type PacketReadWriter interface {
	Wrap
	BasePacketWriter
	BasePacketReader
}

type PacketReadWriteCloser interface {
	Wrap
	BasePacketWriter
	BasePacketReader
	io.Closer
}
