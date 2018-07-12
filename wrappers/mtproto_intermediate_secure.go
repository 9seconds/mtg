package wrappers

import (
	"bytes"
	"encoding/binary"
	"math/rand"

	"github.com/9seconds/mtg/mtproto"
)

type MTProtoIntermediateSecure struct {
	MTProtoIntermediate
}

func (m *MTProtoIntermediateSecure) Read() ([]byte, error) {
	data, err := m.MTProtoIntermediate.Read()
	if err != nil {
		return nil, err
	}
	length := len(data) - (len(data) % 4)

	return data[:length], nil
}

func (m *MTProtoIntermediateSecure) Write(p []byte) (int, error) {
	defer func() {
		m.writeCounter++
	}()

	m.logger.Debugw("Write packet",
		"simple_ack", m.opts.WriteHacks.SimpleAck,
		"quick_ack", m.opts.WriteHacks.QuickAck,
		"counter", m.writeCounter,
	)

	if m.opts.ReadHacks.SimpleAck {
		return m.conn.Write(p)
	}

	buf := &bytes.Buffer{}
	paddingLength := rand.Intn(4)
	buf.Grow(4 + len(p) + paddingLength)

	binary.Write(buf, binary.LittleEndian, uint32(len(p)+paddingLength))
	buf.Write(p)
	buf.Write(make([]byte, paddingLength))

	m.logger.Debugw("Write packet with padding",
		"simple_ack", m.opts.WriteHacks.SimpleAck,
		"quick_ack", m.opts.WriteHacks.QuickAck,
		"counter", m.writeCounter,
		"padding_length", paddingLength,
		"length", len(p),
	)

	_, err := m.conn.Write(buf.Bytes())

	return len(p), err
}

func NewMTProtoIntermediateSecure(conn StreamReadWriteCloser, opts *mtproto.ConnectionOpts) PacketReadWriteCloser {
	return &MTProtoIntermediateSecure{
		MTProtoIntermediate: MTProtoIntermediate{
			conn:   conn,
			logger: conn.Logger().Named("mtproto-intermediate-secure"),
			opts:   opts,
		},
	}
}
