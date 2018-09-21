package rpc

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
)

// ProxyRequest is the data type for storing data required to compose
// RPC_PROXY_REQ request.
type ProxyRequest struct {
	Flags        proxyRequestFlags
	ConnectionID []byte
	OurIPPort    []byte
	ClientIPPort []byte
	ADTag        []byte
	Options      *mtproto.ConnectionOpts
}

// MakeHeader makes RPC_PROXY_REQ header. We need only to append the
// data for it.
func (r *ProxyRequest) MakeHeader(message []byte) (*bytes.Buffer, fmt.Stringer) {
	bufferLength := len(TagProxyRequest) +
		4 + // len(flags)
		len(r.ConnectionID) +
		len(r.ClientIPPort) +
		len(r.OurIPPort) +
		len(ProxyRequestExtraSize) +
		len(ProxyRequestProxyTag) +
		1 + // len(AdTag)
		len(r.ADTag)
	bufferLength += bufferLength % 4

	buf := &bytes.Buffer{}
	buf.Grow(bufferLength + len(message))

	flags := r.Flags
	if r.Options.ReadHacks.QuickAck {
		flags |= proxyRequestFlagsQuickAck
	}

	if bytes.HasPrefix(message, proxyRequestFlagsEncryptedPrefix[:]) {
		flags |= proxyRequestFlagsEncrypted
	}

	buf.Write(TagProxyRequest)                 // nolint: gosec
	buf.Write(flags.Bytes())                   // nolint: gosec
	buf.Write(r.ConnectionID)                  // nolint: gosec
	buf.Write(r.ClientIPPort)                  // nolint: gosec
	buf.Write(r.OurIPPort)                     // nolint: gosec
	buf.Write(ProxyRequestExtraSize)           // nolint: gosec
	buf.Write(ProxyRequestProxyTag)            // nolint: gosec
	buf.WriteByte(byte(len(r.ADTag)))          // nolint: gosec
	buf.Write(r.ADTag)                         // nolint: gosec
	buf.Write(make([]byte, (4-buf.Len()%4)%4)) // nolint: gosec

	return buf, flags
}

// NewProxyRequest build new ProxyRequest data structure.
func NewProxyRequest(clientAddr, ownAddr *net.TCPAddr,
	opts *mtproto.ConnectionOpts, adTag []byte) (*ProxyRequest, error) {
	flags := proxyRequestFlagsHasAdTag | proxyRequestFlagsMagic | proxyRequestFlagsExtMode2

	switch opts.ConnectionType {
	case mtproto.ConnectionTypeAbridged:
		flags |= proxyRequestFlagsAbdridged
	case mtproto.ConnectionTypeIntermediate:
		flags |= proxyRequestFlagsIntermediate
	case mtproto.ConnectionTypeSecure:
		flags |= proxyRequestFlagsIntermediate | proxyRequestFlagsPad
	default:
		panic("Unknown connection type")
	}

	request := &ProxyRequest{
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

	return request, nil
}
