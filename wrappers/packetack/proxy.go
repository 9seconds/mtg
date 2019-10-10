package packetack

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/hub"
	"github.com/9seconds/mtg/mtproto/rpc"
	"github.com/9seconds/mtg/protocol"
)

type wrapperProxy struct {
	flags        rpc.ProxyRequestFlags
	request      *protocol.TelegramRequest
	clientIPPort []byte
	ourIPPort    []byte
	channelRead  hub.ChannelReadCloser
}

func (w *wrapperProxy) Write(packet conntypes.Packet, acks *conntypes.ConnectionAcks) error {
	buf := bytes.Buffer{}

	flags := w.flags
	if acks.Quick {
		flags |= rpc.ProxyRequestFlagsQuickAck
	}
	if bytes.HasPrefix(packet, rpc.ProxyRequestFlagsEncryptedPrefix[:]) {
		flags |= rpc.ProxyRequestFlagsEncrypted
	}

	buf.Write(rpc.TagProxyRequest)
	buf.Write(flags.Bytes())
	buf.Write(w.request.ConnID[:])
	buf.Write(w.clientIPPort)
	buf.Write(w.ourIPPort)
	buf.Write(rpc.ProxyRequestExtraSize)
	buf.Write(rpc.ProxyRequestProxyTag)
	buf.WriteByte(byte(len(config.C.AdTag)))
	buf.Write(config.C.AdTag)
	buf.Write(make([]byte, (4-buf.Len()%4)%4))
	buf.Write(packet)

	return hub.Hub.Write(buf.Bytes(), w.request)
}

func (w *wrapperProxy) Read(acks *conntypes.ConnectionAcks) (conntypes.Packet, error) {
	resp, err := w.channelRead.Read()
	if err != nil {
		return nil, fmt.Errorf("cannot read a response: %w", err)
	}

	if resp.Type == rpc.ProxyResponseTypeSimpleAck {
		acks.Simple = true
	}

	return resp.Payload, nil
}

func (w *wrapperProxy) Close() error {
	return w.channelRead.Close()
}

func NewProxy(request *protocol.TelegramRequest) conntypes.PacketAckReadWriteCloser {
	flags := rpc.ProxyRequestFlagsHasAdTag | rpc.ProxyRequestFlagsMagic | rpc.ProxyRequestFlagsExtMode2

	switch request.ClientProtocol.ConnectionType() {
	case conntypes.ConnectionTypeAbridged:
		flags |= rpc.ProxyRequestFlagsAbdridged
	case conntypes.ConnectionTypeIntermediate:
		flags |= rpc.ProxyRequestFlagsIntermediate
	case conntypes.ConnectionTypeSecure:
		flags |= rpc.ProxyRequestFlagsIntermediate | rpc.ProxyRequestFlagsPad
	default:
		panic("unknown connection type")
	}

	return &wrapperProxy{
		flags:        flags,
		request:      request,
		channelRead:  hub.Registry.Register(request.ConnID),
		clientIPPort: proxyGetIPPort(request.ClientConn.RemoteAddr()),
		ourIPPort:    proxyGetIPPort(request.ClientConn.LocalAddr()),
	}
}

func proxyGetIPPort(addr *net.TCPAddr) []byte {
	rv := [16 + 4]byte{}
	port := [4]byte{}

	copy(rv[:16], addr.IP.To16())
	binary.LittleEndian.PutUint32(port[:], uint32(addr.Port))
	copy(rv[16:], port[:])

	return rv[:]
}
