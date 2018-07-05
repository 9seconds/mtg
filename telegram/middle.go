package telegram

import (
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/juju/errors"
	"go.uber.org/zap"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/mtproto/rpc"
	mtwrappers "github.com/9seconds/mtg/mtproto/wrappers"
	"github.com/9seconds/mtg/wrappers"
)

type middleTelegram struct {
	middleTelegramCaller

	conf *config.Config
}

func NewMiddleTelegram(conf *config.Config, logger *zap.SugaredLogger) Telegram {
	tg := &middleTelegram{
		middleTelegramCaller: middleTelegramCaller{
			baseTelegram: baseTelegram{
				dialer: tgDialer{
					Dialer: net.Dialer{Timeout: telegramDialTimeout},
					conf:   conf,
				},
			},
			logger: logger,
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

func (t *middleTelegram) Init(connOpts *mtproto.ConnectionOpts, conn wrappers.ReadWriteCloserWithAddr) (wrappers.ReadWriteCloserWithAddr, error) {
	rpcNonceConn := mtwrappers.NewFrameRWC(conn, rpc.SeqNoNonce)

	rpcNonceReq, err := t.sendRPCNonceRequest(rpcNonceConn)
	if err != nil {
		return nil, err
	}
	rpcNonceResp, err := t.receiveRPCNonceResponse(rpcNonceConn, rpcNonceReq)
	if err != nil {
		return nil, err
	}

	secureConn := mtwrappers.NewMiddleProxyCipherRWC(conn, rpcNonceReq, rpcNonceResp, t.proxySecret)
	secureConn = mtwrappers.NewFrameRWC(secureConn, rpc.SeqNoHandshake)

	rpcHandshakeReq, err := t.sendRPCHandshakeRequest(secureConn)
	if err != nil {
		return nil, err
	}
	_, err = t.receiveRPCHandshakeResponse(secureConn, rpcHandshakeReq)
	if err != nil {
		return nil, err
	}

	return mtwrappers.NewProxyRequestRWC(secureConn, connOpts, t.conf.AdTag)
}

func (t *middleTelegram) sendRPCNonceRequest(conn io.Writer) (*rpc.NonceRequest, error) {
	rpcNonceReq, err := rpc.NewNonceRequest(t.proxySecret)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create RPC nonce request")
	}
	if _, err = conn.Write(rpcNonceReq.Bytes()); err != nil {
		return nil, errors.Annotate(err, "Cannot send RPC nonce request")
	}

	return rpcNonceReq, nil
}

func (t *middleTelegram) receiveRPCNonceResponse(conn io.Reader, req *rpc.NonceRequest) (*rpc.NonceResponse, error) {
	var ans [128]byte

	n, err := conn.Read(ans[:])
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read RPC nonce response")
	}

	rpcNonceResp, err := rpc.NewNonceResponse(ans[:n])
	if err != nil {
		return nil, errors.Annotate(err, "Cannot initialize RPC nonce response")
	}
	if err = rpcNonceResp.Valid(req); err != nil {
		return nil, errors.Annotate(err, "Invalid RPC nonce response")
	}

	return rpcNonceResp, nil
}

func (t *middleTelegram) sendRPCHandshakeRequest(conn io.Writer) (*rpc.HandshakeRequest, error) {
	req := rpc.NewHandshakeRequest()
	if _, err := conn.Write(req.Bytes()); err != nil {
		return nil, errors.Annotate(err, "Cannot send RPC handshake request")
	}

	return req, nil
}

func (t *middleTelegram) receiveRPCHandshakeResponse(conn io.Reader, req *rpc.HandshakeRequest) (*rpc.HandshakeResponse, error) {
	var ans [128]byte

	n, err := conn.Read(ans[:])
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read RPC handshake response")
	}

	rpcHandshakeResp, err := rpc.NewHandshakeResponse(ans[:n])
	if err != nil {
		return nil, errors.Annotate(err, "Cannot initialize RPC handshake response")
	}
	if err = rpcHandshakeResp.Valid(req); err != nil {
		return nil, errors.Annotate(err, "Invalid RPC handshake response")
	}

	return rpcHandshakeResp, nil
}
