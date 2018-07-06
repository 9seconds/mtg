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
	req  *rpc.ProxyRequest
}

func (p *ProxyRequestReadWriteCloserWithAddr) Read(buf []byte) (int, error) {
	return p.BufferedRead(buf, func() error {
		ans := make([]byte, 4)
		if _, err := io.ReadFull(p.conn, ans); err != nil {
			return errors.Annotate(err, "Cannot read RPC tag")
		}

		switch {
		case bytes.Equal(ans, rpc.TagProxyAns):
			return p.readProxyAns(buf)
		case bytes.Equal(ans, rpc.TagSimpleAck):
			return p.readSimpleAck()
		case bytes.Equal(ans, rpc.TagCloseExt):
			return p.readCloseExt()
		}

		return errors.Errorf("Unknown RPC answer %v", ans)
	})
}

func (p *ProxyRequestReadWriteCloserWithAddr) readCloseExt() error {
	return errors.New("Connection has been closed remotely")
}

func (p *ProxyRequestReadWriteCloserWithAddr) readProxyAns(buf []byte) (err error) {
	if _, err = io.CopyN(ioutil.Discard, p.conn, 8+4); err != nil {
		return errors.Annotate(err, "Cannot skip flags and connid")
	}

	n := len(buf)
	preBuffer := &bytes.Buffer{}
	for n == len(buf) {
		n, err = p.conn.Read(buf)
		if err != nil {
			return errors.Annotate(err, "Cannot read proxy answer")
		}
		preBuffer.Write(buf[:n])
	}
	p.Buffer.Write(preBuffer.Bytes())

	return nil
}

func (p *ProxyRequestReadWriteCloserWithAddr) readSimpleAck() error {
	if _, err := io.CopyN(ioutil.Discard, p.conn, 8); err != nil {
		return errors.Annotate(err, "Cannot skip connid")
	}

	ackData := make([]byte, 4)
	if _, err := io.ReadFull(p.conn, ackData); err != nil {
		return errors.Annotate(err, "Cannot read simple ack")
	}
	p.Buffer.Write(ackData)

	return nil
}

func (p *ProxyRequestReadWriteCloserWithAddr) Write(raw []byte) (int, error) {
	if _, err := p.conn.Write(p.req.Bytes(raw)); err != nil {
		return 0, err
	}

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
	req, err := rpc.NewProxyRequest(connOpts.ClientAddr, conn.LocalAddr(), connOpts, adTag)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create new RPC proxy request")
	}

	return &ProxyRequestReadWriteCloserWithAddr{
		BufferedReader: wrappers.NewBufferedReader(),
		conn:           conn,
		req:            req,
	}, nil
}
