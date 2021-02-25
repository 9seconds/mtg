package mtproto

import (
	"fmt"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/mtproto/rpc"
	"github.com/9seconds/mtg/protocol"
	"github.com/9seconds/mtg/telegram"
	"github.com/9seconds/mtg/wrappers/packet"
	"github.com/9seconds/mtg/wrappers/stream"
)

func TelegramProtocol(req *protocol.TelegramRequest) (conntypes.PacketReadWriteCloser, error) {
	conn, err := telegram.Middle.Dial(req.ClientProtocol.DC(),
		req.ClientProtocol.ConnectionProtocol())
	if err != nil {
		return nil, fmt.Errorf("cannot connect to telegram: %w", err)
	}

	rpcNonceConn := packet.NewMtprotoFrame(conn, rpc.SeqNoNonce)

	rpcNonceReq, err := doRPCNonceRequest(rpcNonceConn)
	if err != nil {
		return nil, fmt.Errorf("cannot do nonce request: %w", err)
	}

	rpcNonceResp, err := getRPCNonceResponse(rpcNonceConn, rpcNonceReq)
	if err != nil {
		return nil, fmt.Errorf("cannot get nonce response: %w", err)
	}

	secureConn := stream.NewMiddleProxyCipher(conn, rpcNonceReq, rpcNonceResp, telegram.Middle.Secret())
	frameConn := packet.NewMtprotoFrame(secureConn, rpc.SeqNoHandshake)

	if err := doRPCHandshakeRequest(frameConn); err != nil {
		return nil, fmt.Errorf("cannot do handshake request: %w", err)
	}

	if err := getRPCHandshakeResponse(frameConn); err != nil {
		return nil, fmt.Errorf("cannot get handshake response: %w", err)
	}

	return frameConn, nil
}

func doRPCNonceRequest(conn conntypes.BasePacketWriter) (*rpc.NonceRequest, error) {
	rpcNonceReq, err := rpc.NewNonceRequest(telegram.Middle.Secret())
	if err != nil {
		panic(err)
	}

	if err := conn.Write(rpcNonceReq.Bytes()); err != nil {
		return nil, err // nolint: wrapcheck
	}

	return rpcNonceReq, nil
}

func getRPCNonceResponse(conn conntypes.BasePacketReader, req *rpc.NonceRequest) (*rpc.NonceResponse, error) {
	packet, err := conn.Read()
	if err != nil {
		return nil, fmt.Errorf("cannot read from connection: %w", err)
	}

	resp, err := rpc.NewNonceResponse(packet)
	if err != nil {
		return nil, fmt.Errorf("cannot build rpc nonce response: %w", err)
	}

	if err = resp.Valid(req); err != nil {
		return nil, fmt.Errorf("invalid nonce response: %w", err)
	}

	return resp, nil
}

func doRPCHandshakeRequest(conn conntypes.BasePacketWriter) error {
	if err := conn.Write(rpc.HandshakeRequest); err != nil {
		return fmt.Errorf("cannot make a request: %w", err)
	}

	return nil
}

func getRPCHandshakeResponse(conn conntypes.BasePacketReader) error {
	packet, err := conn.Read()
	if err != nil {
		return fmt.Errorf("cannot read a response: %w", err)
	}

	resp, err := rpc.NewHandshakeResponse(packet)
	if err != nil {
		return fmt.Errorf("cannot build a handshake response: %w", err)
	}

	if err := resp.Valid(); err != nil {
		return fmt.Errorf("invalid handshake response: %w", err)
	}

	return nil
}
