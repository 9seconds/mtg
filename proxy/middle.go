package proxy

import (
	"context"
	"net"
	"sync"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/client"
	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/telegram"
	"github.com/9seconds/mtg/wrappers"
)

func NewProxyMiddle(conf *config.Config) *Proxy {
	tg := telegram.NewMiddleTelegram(conf)

	return &Proxy{
		conf: conf,
		acceptCallback: func(ctx context.Context, cancel context.CancelFunc, clientSocket net.Conn,
			connID string, wait *sync.WaitGroup, conf *config.Config) error {
			client, opts, err := client.MiddleInit(ctx, cancel, clientSocket, connID, conf)
			if err != nil {
				return errors.Annotate(err, "Cannot initialize client connection")
			}
			defer client.Close()

			server, err := middleTelegramStream(ctx, cancel, opts, connID, tg)
			if err != nil {
				return errors.Annotate(err, "Cannot initialize telegram connection")
			}
			defer server.Close()

			wait.Add(2)

			go middlePipe(client, server, wait, &opts.ReadHacks)
			go middlePipe(server, client, wait, &opts.WriteHacks)

			return nil
		},
	}
}

func middleTelegramStream(ctx context.Context, cancel context.CancelFunc, opts *mtproto.ConnectionOpts,
	connID string, tg *telegram.MiddleTelegram) (wrappers.WrapPacketReadWriteCloser, error) {
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

func middlePipe(src wrappers.WrapPacketReader, dst wrappers.WrapPacketWriter, wait *sync.WaitGroup, hacks *mtproto.Hacks) {
	defer wait.Done()

	for {
		hacks.SimpleAck = false
		hacks.QuickAck = false

		packet, err := src.Read()
		if err != nil {
			return
		}
		if _, err = dst.Write(packet); err != nil {
			return
		}
	}
}
