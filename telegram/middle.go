package telegram

import (
	"io/ioutil"
	"net"
	"net/http"
	"sync"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/mtproto/rpc"
	mtwrappers "github.com/9seconds/mtg/mtproto/wrappers"
	"github.com/9seconds/mtg/wrappers"
	"github.com/juju/errors"
)

type middleTelegram struct {
	middleTelegramCaller
}

func NewMiddleTelegram(conf *config.Config, logger *zap.SugaredLogger) Telegram {
	tg := &middleTelegram{
		middleTelegramCaller: middleTelegramCaller{
			baseTelegram: baseTelegram{
				dialer: tgDialer{net.Dialer{Timeout: telegramDialTimeout}},
			},
			logger: logger,
			httpClient: &http.Client{
				Timeout: middleTelegramHTTPClientTimeout,
			},
			dialerMutex: &sync.RWMutex{},
		},
	}

	if err := tg.update(); err != nil {
		panic(err)
	}
	go tg.autoUpdate()

	return tg
}

func (t *middleTelegram) Init(connOpts *mtproto.ConnectionOpts, conn wrappers.ReadWriteCloserWithAddr) (wrappers.ReadWriteCloserWithAddr, error) {
	rpcNonceConn := mtwrappers.NewFrameRWC(conn, rpc.RPCNonceSeqNo)

	rpcNonceReq, err := t.sendRPCNonceRequest(rpcNonceConn)
	if err != nil {
		return nil, err
	}
	rpcNonceResp, err := t.receiveRPCNonceResponse(rpcNonceConn, rpcNonceReq)
	if err != nil {
		return nil, err
	}

	secureConn := mtwrappers.NewFrameRWC(conn, rpc.RPCHandshakeSeqNo)
	secureConn = mtwrappers.NewMiddleProxyCipherRWC(secureConn, rpcNonceReq,
		rpcNonceResp, connOpts.ClientAddr, t.proxySecret)

	rpcHandshakeReq, err := t.sendRPCHandshakeRequest(secureConn)
	if err != nil {
		return nil, err
	}
	_, err = t.receiveRPCHandshakeResponse(secureConn, rpcHandshakeReq)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (t *middleTelegram) sendRPCNonceRequest(conn wrappers.ReadWriteCloserWithAddr) (*rpc.RPCNonceRequest, error) {
	rpcNonceReq, err := rpc.NewRPCNonceRequest(t.proxySecret)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create RPC nonce request")
	}
	if _, err = conn.Write(rpcNonceReq.Bytes()); err != nil {
		return nil, errors.Annotate(err, "Cannot send RPC nonce request")
	}

	return rpcNonceReq, nil
}

func (t *middleTelegram) receiveRPCNonceResponse(conn wrappers.ReadWriteCloserWithAddr, req *rpc.RPCNonceRequest) (*rpc.RPCNonceResponse, error) {
	ans, err := ioutil.ReadAll(conn)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read RPC nonce response")
	}
	rpcNonceResp, err := rpc.NewRPCNonceResponse(ans)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot initialize RPC nonce response")
	}
	if err = rpcNonceResp.Valid(req); err != nil {
		return nil, errors.Annotate(err, "Invalid RPC nonce response")
	}

	return rpcNonceResp, nil
}

func (t *middleTelegram) sendRPCHandshakeRequest(conn wrappers.ReadWriteCloserWithAddr) (*rpc.RPCHandshakeRequest, error) {
	req := rpc.NewRPCHandshakeRequest()
	if _, err := conn.Write(req.Bytes()); err != nil {
		return nil, errors.Annotate(err, "Cannot send RPC handshake request")
	}

	return req, nil
}

func (t *middleTelegram) receiveRPCHandshakeResponse(conn wrappers.ReadWriteCloserWithAddr, req *rpc.RPCHandshakeRequest) (*rpc.RPCHandshakeResponse, error) {
	ans, err := ioutil.ReadAll(conn)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read RPC handshake response")
	}
	rpcHandshakeResp, err := rpc.NewRPCHandshakeResponse(ans)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot initialize RPC handshake response")
	}
	if err = rpcHandshakeResp.Valid(req); err != nil {
		return nil, errors.Annotate(err, "Invalid RPC handshake response")
	}

	return rpcHandshakeResp, nil
}
