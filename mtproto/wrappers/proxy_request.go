package wrappers

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/mtproto/rpc"
	"github.com/9seconds/mtg/wrappers"
)

type ProxyRequestReadWriteCloserWithAddr struct {
	wrappers.BufferedReader

	conn wrappers.ReadWriteCloserWithAddr
	req  *rpc.RPCProxyRequest
}

func (p *ProxyRequestReadWriteCloserWithAddr) Read(buf []byte) (int, error) {
	return p.BufferedRead(buf, func() error {
		ansBuf := &bytes.Buffer{}
		ansBuf.Grow(4)

		if _, err := io.CopyN(ansBuf, p.conn, 4); err != nil {
			return errors.Annotate(err, "Cannot read RPC tag")
		}

		if bytes.Equal(ansBuf.Bytes(), rpc.RPCTagCloseExt) {
			return p.readCloseExt()
		} else if bytes.Equal(ansBuf.Bytes(), rpc.RPCTagProxyAns) {
			return p.readProxyAns(buf)
		} else if bytes.Equal(ansBuf.Bytes(), rpc.RPCTagSimpleAck) {
			return p.readSimpleAck()
		}

		return nil
	})
}

func (p *ProxyRequestReadWriteCloserWithAddr) readCloseExt() error {
	return errors.New("Connection has been closed remotely")
}

func (p *ProxyRequestReadWriteCloserWithAddr) readProxyAns(buf []byte) error {
	if _, err := io.CopyN(ioutil.Discard, p.conn, 8+4); err != nil {
		return errors.Annotate(err, "Cannot skip flags and connid")
	}

	for {
		n, err := p.conn.Read(buf)
		if err != nil {
			return errors.Annotate(err, "Cannot read proxy answer")
		}
		if n == 0 {
			break
		}
		p.Buffer.Write(buf[:n])
	}

	return nil
}

func (p *ProxyRequestReadWriteCloserWithAddr) readSimpleAck() error {
	if _, err := io.CopyN(ioutil.Discard, p.conn, 8); err != nil {
		return errors.Annotate(err, "Cannot skip connid")
	}
	if _, err := io.CopyN(p.Buffer, p.conn, 4); err != nil {
		return errors.Annotate(err, "Cannot read simple ack")
	}
	p.req.Options.SimpleAck = true

	return nil
}

func (p *ProxyRequestReadWriteCloserWithAddr) Write(raw []byte) (int, error) {
	if _, err := p.conn.Write(p.req.Bytes(raw)); err != nil {
		return 0, err
	}
	p.req.Options.SimpleAck = false
	p.req.Options.QuickAck = false

	return len(raw), nil
}

func (p *ProxyRequestReadWriteCloserWithAddr) Close() error {
	return p.conn.Close()
}

func (p *ProxyRequestReadWriteCloserWithAddr) LocalAddr() *net.TCPAddr {
	return p.conn.LocalAddr()
}

func (p *ProxyRequestReadWriteCloserWithAddr) RemoteAddr() *net.TCPAddr {
	return p.conn.RemoteAddr()
}

func NewProxyRequestRWC(conn wrappers.ReadWriteCloserWithAddr, connOpts *mtproto.ConnectionOpts, adTag []byte) (wrappers.ReadWriteCloserWithAddr, error) {
	req, err := rpc.NewRPCProxyRequest(connOpts.ClientAddr, conn.LocalAddr(), connOpts, adTag)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create new RPC proxy request")
	}

	return &ProxyRequestReadWriteCloserWithAddr{
		conn: conn,
		req:  req,
	}, nil
}
