package wrappers

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

const intermediateQuickAckLength = 0x80000000

type IntermediateReadWriteCloserWithAddr struct {
	wrappers.BufferedReader

	conn wrappers.ReadWriteCloserWithAddr
	opts *mtproto.ConnectionOpts
}

func (i *IntermediateReadWriteCloserWithAddr) Read(p []byte) (int, error) {
	return i.BufferedRead(p, func() error {
		i.opts.QuickAck = false
		i.opts.SimpleAck = false

		buf := &bytes.Buffer{}
		buf.Grow(4)

		if _, err := io.CopyN(buf, i.conn, 4); err != nil {
			return errors.Annotate(err, "Cannot read message length")
		}
		length := binary.LittleEndian.Uint32(buf.Bytes())
		buf.Reset()
		buf.Grow(int(length))

		if length > intermediateQuickAckLength {
			i.opts.QuickAck = true
			length -= intermediateQuickAckLength
		}

		if _, err := io.CopyN(buf, i.conn, int64(length)); err != nil {
			return errors.Annotate(err, "Cannot read the message")
		}

		if length%4 != 0 {
			length -= length % 4
			i.Buffer.Write(buf.Bytes()[:length])
			return nil
		}

		i.Buffer.Write(buf.Bytes())

		return nil
	})
}

func (i *IntermediateReadWriteCloserWithAddr) Write(p []byte) (int, error) {
	if i.opts.SimpleAck {
		return i.conn.Write(p)
	}

	var length [4]byte
	binary.LittleEndian.PutUint32(length[:], uint32(len(p)))

	return i.conn.Write(append(length[:], p...))
}

func (i *IntermediateReadWriteCloserWithAddr) Close() error {
	return i.conn.Close()
}

func (i *IntermediateReadWriteCloserWithAddr) LocalAddr() *net.TCPAddr {
	return i.conn.LocalAddr()
}

func (i *IntermediateReadWriteCloserWithAddr) RemoteAddr() *net.TCPAddr {
	return i.conn.RemoteAddr()
}

func NewIntermediateRWC(conn wrappers.ReadWriteCloserWithAddr, connOpts *mtproto.ConnectionOpts) wrappers.ReadWriteCloserWithAddr {
	return &IntermediateReadWriteCloserWithAddr{
		BufferedReader: wrappers.NewBufferedReader(),
		conn:           conn,
		opts:           connOpts,
	}
}
