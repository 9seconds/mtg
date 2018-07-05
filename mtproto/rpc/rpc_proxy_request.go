package rpc

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
)

type RPCProxyRequest struct {
	Flags        RPCProxyRequestFlags
	ConnectionID []byte
	OurIPPort    []byte
	ClientIPPort []byte
	ADTag        []byte
	Options      *mtproto.ConnectionOpts
}

func (r *RPCProxyRequest) Bytes(message []byte) []byte {
	buf := &bytes.Buffer{}

	flags := r.Flags
	if r.Options.QuickAck {
		flags |= RPCProxyRequestFlagsQuickAck
	}

	if bytes.HasPrefix(message, rpcProxyRequestFlagsEncryptedPrefix[:]) {
		flags |= RPCProxyRequestFlagsEncrypted
	}

	buf.Write(RPCTagProxyRequest)
	buf.Write(flags.Bytes())
	buf.Write(r.ConnectionID[:])
	buf.Write(r.ClientIPPort[:])
	buf.Write(r.OurIPPort[:])
	buf.Write(RPCProxyRequestExtraSize)
	buf.Write(RPCProxyRequestProxyTag)
	buf.WriteByte(byte(len(r.ADTag)))
	buf.Write(r.ADTag)
	buf.Write(bytes.Repeat([]byte{0x00}, buf.Len()%4))
	buf.Write(message)

	return buf.Bytes()
}

func NewRPCProxyRequest(clientAddr, ownAddr *net.TCPAddr, opts *mtproto.ConnectionOpts, adTag []byte) (*RPCProxyRequest, error) {
	flags := RPCProxyRequestFlagsHasAdTag | RPCProxyRequestFlagsMagic | RPCProxyRequestFlagsExtMode2

	switch opts.ConnectionType {
	case mtproto.ConnectionTypeAbridged:
		flags |= RPCProxyRequestFlagsAbdridged
	case mtproto.ConnectionTypeIntermediate:
		flags |= RPCProxyRequestFlagsIntermediate
	}

	request := RPCProxyRequest{
		Flags:        flags,
		ADTag:        adTag,
		Options:      opts,
		ConnectionID: make([]byte, 8),
		ClientIPPort: make([]byte, 16+4),
		OurIPPort:    make([]byte, 16+4),
	}

	if _, err := rand.Read(request.ConnectionID); err != nil {
		return nil, errors.Annotate(err, "Cannot generate connection ID")
	}

	port := [4]byte{}
	copy(request.ClientIPPort[:16], clientAddr.IP.To16())
	binary.LittleEndian.PutUint32(port[:], uint32(clientAddr.Port))
	copy(request.ClientIPPort[16:], port[:])

	copy(request.OurIPPort[:16], ownAddr.IP.To16())
	binary.LittleEndian.PutUint32(port[:], uint32(ownAddr.Port))
	copy(request.OurIPPort[16:], port[:])

	return &request, nil
}
