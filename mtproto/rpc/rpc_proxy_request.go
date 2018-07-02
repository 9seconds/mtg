package rpc

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"net"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/mtproto/bufferpool"
	"github.com/juju/errors"
)

const (
	rpcProxyRequestConnectionIDLength = 8
	rpcProxyRequestIPPortLength       = 16 + 4
)

var (
	rpcProxyRequestTag       = []byte{0xee, 0xf1, 0xce, 0x36}
	rpcProxyRequestExtraSize = []byte{0x18, 0x00, 0x00, 0x00}
	rpcProxyRequestProxyTag  = []byte{0xae, 0x26, 0x1e, 0xdb}
)

type RPCProxyRequest struct {
	Flags        RPCProxyRequestFlags
	ConnectionID [rpcProxyRequestConnectionIDLength]byte
	RemoteIPPort [rpcProxyRequestIPPortLength]byte
	LocalIPPort  [rpcProxyRequestIPPortLength]byte
	ADTag        []byte
	Message      *bytes.Buffer
	Extras       *mtproto.Extras
}

func (r *RPCProxyRequest) Bytes() *bytes.Buffer {
	buf := bufferpool.Get()

	flags := r.Flags
	if r.Extras.QuickAck {
		flags |= RPCProxyRequestFlagsQuickAck
	}

	messageBytes := r.Message.Bytes()
	if bytes.HasPrefix(messageBytes, rpcProxyRequestFlagsEncryptedPrefix[:]) {
		flags |= RPCProxyRequestFlagsEncrypted
	}

	buf.Write(rpcProxyRequestTag)
	buf.Write(flags.Bytes())
	buf.Write(r.ConnectionID[:])
	buf.Write(r.RemoteIPPort[:])
	buf.Write(r.LocalIPPort[:])
	buf.Write(rpcProxyRequestExtraSize)
	buf.Write(rpcProxyRequestProxyTag)
	buf.WriteByte(byte(len(r.ADTag)))
	buf.Write(r.ADTag)

	for i := 0; i < (buf.Len() % 4); i++ {
		buf.WriteByte(0x00)
	}
	if r.Message != nil {
		buf.Write(messageBytes)
	}

	return buf
}

func NewRPCProxyRequest(connectionType mtproto.ConnectionType, local, remote *net.TCPAddr, adTag []byte, extras *mtproto.Extras) (*RPCProxyRequest, error) {
	flags := RPCProxyRequestFlagsHasAdTag | RPCProxyRequestFlagsMagic | RPCProxyRequestFlagsExtMode2

	switch connectionType {
	case mtproto.ConnectionTypeAbridged:
		flags |= RPCProxyRequestFlagsAbdridged
	case mtproto.ConnectionTypeIntermediate:
		flags |= RPCProxyRequestFlagsIntermediate
	}

	request := RPCProxyRequest{
		Flags:  flags,
		ADTag:  adTag,
		Extras: extras,
	}

	if _, err := rand.Read(request.ConnectionID[:]); err != nil {
		return nil, errors.Annotate(err, "Cannot generate connection ID")
	}

	port := make([]byte, 4)
	copy(request.LocalIPPort[:], local.IP.To16())
	binary.LittleEndian.PutUint32(port, uint32(local.Port))
	copy(request.LocalIPPort[16:], port)

	copy(request.RemoteIPPort[:], remote.IP.To16())
	binary.LittleEndian.PutUint32(port, uint32(remote.Port))
	copy(request.RemoteIPPort[16:], port)

	return &request, nil
}
