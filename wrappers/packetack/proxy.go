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
	request      *protocol.TelegramRequest
	proxy        *hub.ProxyConn
	clientIPPort []byte
	ourIPPort    []byte
	flags        rpc.ProxyRequestFlags
}

func (w *wrapperProxy) Write(packet conntypes.Packet, acks *conntypes.ConnectionAcks) error {
	buf := acquireProxyBytesBuffer()
	defer releaseProxyBytesBuffer(buf)

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

	return w.proxy.Write(buf.Bytes())
}

func (w *wrapperProxy) Read(acks *conntypes.ConnectionAcks) (conntypes.Packet, error) {
	resp, err := w.proxy.Read()
	if err != nil {
		return nil, fmt.Errorf("cannot read a response: %w", err)
	}

	if resp.Type == rpc.ProxyResponseTypeSimpleAck {
		acks.Simple = true
	}

	return resp.Payload, nil
}

func (w *wrapperProxy) Close() error {
	w.proxy.Close()
	return nil
}

func NewProxy(request *protocol.TelegramRequest) (conntypes.PacketAckReadWriteCloser, error) {
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

	proxy, err := hub.Hub.Register(request)
	if err != nil {
		return nil, fmt.Errorf("cannot make a new proxy wrapper: %w", err)
	}

	return &wrapperProxy{
		flags:        flags,
		request:      request,
		proxy:        proxy,
		clientIPPort: proxyGetIPPort(request.ClientConn.RemoteAddr()),
		ourIPPort:    proxyGetIPPort(request.ClientConn.LocalAddr()),
	}, nil
}

func proxyGetIPPort(addr *net.TCPAddr) []byte {
	rv := [16 + 4]byte{}
	port := [4]byte{}

	copy(rv[:16], addr.IP.To16())
	binary.LittleEndian.PutUint32(port[:], uint32(addr.Port))
	copy(rv[16:], port[:])

	return rv[:]
}
