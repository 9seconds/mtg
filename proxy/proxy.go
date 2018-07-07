package proxy

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/juju/errors"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"

	"github.com/9seconds/mtg/config"
)

type proxyAcceptCallback func(context.Context, context.CancelFunc, net.Conn, string, *sync.WaitGroup, *config.Config) (io.Closer, io.Closer, error)

type Proxy struct {
	conf           *config.Config
	acceptCallback proxyAcceptCallback
}

func (p *Proxy) Serve() error {
	lsock, err := net.Listen("tcp", p.conf.BindAddr())
	if err != nil {
		return errors.Annotate(err, "Cannot create listen socket")
	}

	for {
		if conn, err := lsock.Accept(); err != nil {
			zap.S().Errorw("Cannot allocate incoming connection", "error", err)
		} else {
			go p.accept(conn)
		}
	}
}

func (p *Proxy) accept(conn net.Conn) {
	connID := uuid.NewV4().String()
	log := zap.S().With("connection_id", connID)

	defer func() {
		conn.Close()

		if err := recover(); err != nil {
			log.Errorw("Crash of accept handler", "error", err)
		}
	}()

	log.Infow("Client connected", "addr", conn.RemoteAddr())

	ctx, cancel := context.WithCancel(context.Background())
	wait := &sync.WaitGroup{}

	client, server, err := p.acceptCallback(ctx, cancel, conn, connID, wait, p.conf)
	defer func() {
		if client != nil {
			client.Close()
		}
		if server != nil {
			server.Close()
		}
	}()
	if err != nil {
		log.Errorw("Cannot initialize connection", "error", err)
		cancel()
	}

	<-ctx.Done()
	wait.Wait()

	log.Infow("Client disconnected", "addr", conn.RemoteAddr())
}
