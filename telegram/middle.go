package telegram

import (
	"net"
	"net/http"
	"sync"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/mtproto/rpc"
	"github.com/9seconds/mtg/wrappers"
)

type MiddleTelegram struct {
	middleTelegramCaller

	conf *config.Config
}

func (t *MiddleTelegram) Init(connOpts *mtproto.ConnectionOpts, conn wrappers.StreamReadWriteCloser) (wrappers.Wrap, error) {
	rpcNonceConn := wrappers.NewMTProtoFrame(conn, rpc.SeqNoNonce)

	rpcNonceReq, err := t.sendRPCNonceRequest(rpcNonceConn)
	if err != nil {
		return nil, err
	}
	rpcNonceResp, err := t.receiveRPCNonceResponse(rpcNonceConn, rpcNonceReq)
	if err != nil {
		return nil, err
	}

	secureConn := wrappers.NewMiddleProxyCipher(conn, rpcNonceReq, rpcNonceResp, t.proxySecret)
	frameConn := wrappers.NewMTProtoFrame(secureConn, rpc.SeqNoHandshake)

	rpcHandshakeReq, err := t.sendRPCHandshakeRequest(frameConn)
	if err != nil {
		return nil, err
	}
	_, err = t.receiveRPCHandshakeResponse(frameConn, rpcHandshakeReq)
	if err != nil {
		return nil, err
	}

	proxyConn, err := wrappers.NewMTProtoProxy(frameConn, connOpts, t.conf.AdTag)
	if err != nil {
		return nil, err
	}
	proxyConn.Logger().Infow("Telegram connection initialized")

	return proxyConn, nil
}

func (t *MiddleTelegram) sendRPCNonceRequest(conn wrappers.PacketWriter) (*rpc.NonceRequest, error) {
	rpcNonceReq, err := rpc.NewNonceRequest(t.proxySecret)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create RPC nonce request")
	}
	if _, err = conn.Write(rpcNonceReq.Bytes()); err != nil {
		return nil, errors.Annotate(err, "Cannot send RPC nonce request")
	}

	return rpcNonceReq, nil
}

func (t *MiddleTelegram) receiveRPCNonceResponse(conn wrappers.PacketReader, req *rpc.NonceRequest) (*rpc.NonceResponse, error) {
	packet, err := conn.Read()
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read RPC nonce response")
	}

	rpcNonceResp, err := rpc.NewNonceResponse(packet)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot initialize RPC nonce response")
	}
	if err = rpcNonceResp.Valid(req); err != nil {
		return nil, errors.Annotate(err, "Invalid RPC nonce response")
	}

	return rpcNonceResp, nil
}

func (t *MiddleTelegram) sendRPCHandshakeRequest(conn wrappers.PacketWriter) (*rpc.HandshakeRequest, error) {
	req := rpc.NewHandshakeRequest()
	if _, err := conn.Write(req.Bytes()); err != nil {
		return nil, errors.Annotate(err, "Cannot send RPC handshake request")
	}

	return req, nil
}

func (t *MiddleTelegram) receiveRPCHandshakeResponse(conn wrappers.PacketReader, req *rpc.HandshakeRequest) (*rpc.HandshakeResponse, error) {
	packet, err := conn.Read()
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read RPC handshake response")
	}

	rpcHandshakeResp, err := rpc.NewHandshakeResponse(packet)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot initialize RPC handshake response")
	}
	if err = rpcHandshakeResp.Valid(req); err != nil {
		return nil, errors.Annotate(err, "Invalid RPC handshake response")
	}

	return rpcHandshakeResp, nil
}

func NewMiddleTelegram(conf *config.Config) Telegram {
	tg := &MiddleTelegram{
		middleTelegramCaller: middleTelegramCaller{
			baseTelegram: baseTelegram{
				dialer: tgDialer{
					Dialer: net.Dialer{Timeout: telegramDialTimeout},
					conf:   conf,
				},
			},
			httpClient: &http.Client{
				Timeout: middleTelegramHTTPClientTimeout,
			},
			dialerMutex: &sync.RWMutex{},
		},
		conf: conf,
	}

	if err := tg.update(); err != nil {
		panic(err)
	}
	go tg.autoUpdate()

	return tg
}
