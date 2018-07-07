package proxy

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/client"
	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/telegram"
	"github.com/9seconds/mtg/wrappers"
)

func NewProxyDirect(conf *config.Config) *Proxy {
	tg := telegram.NewDirectTelegram(conf)

	return &Proxy{
		conf: conf,
		acceptCallback: func(ctx context.Context, cancel context.CancelFunc, clientSocket net.Conn,
			connID string, wait *sync.WaitGroup, conf *config.Config) (io.Closer, io.Closer, error) {
			client, opts, err := client.DirectInit(ctx, cancel, clientSocket, connID, conf)
			if err != nil {
				return nil, nil, errors.Annotate(err, "Cannot initialize client connection")
			}

			server, err := directTelegramStream(ctx, cancel, opts, connID, tg)
			if err != nil {
				return client, nil, errors.Annotate(err, "Cannot initialize telegram connection")
			}

			wait.Add(2)

			go directPipe(client, server, wait)
			go directPipe(server, client, wait)

			return client, server, nil
		},
	}
}

func directTelegramStream(ctx context.Context, cancel context.CancelFunc, opts *mtproto.ConnectionOpts,
	connID string, tg *telegram.DirectTelegram) (wrappers.WrapStreamReadWriteCloser, error) {
	streamConn, err := tg.Dial(connID, opts)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot dial to Telegram")
	}
	streamConn = wrappers.NewCtx(ctx, cancel, streamConn)

	packetConn, err := tg.Init(opts, streamConn)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot handshake telegram")
	}

	return packetConn, nil
}

func directPipe(src io.Reader, dst io.Writer, wait *sync.WaitGroup) {
	defer wait.Done()
	io.Copy(dst, src)
}
