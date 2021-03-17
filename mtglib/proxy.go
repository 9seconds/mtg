package mtglib

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
)

type Proxy struct {
	ctx             context.Context
	ctxCancel       context.CancelFunc
	streamWaitGroup sync.WaitGroup
	workerPool      *ants.PoolWithFunc

	secret          Secret
	network         Network
	antiReplayCache AntiReplayCache
	ipBlocklist     IPBlocklist
	eventStream     EventStream
	logger          Logger
}

func (p *Proxy) ServeConn(conn net.Conn) {
	ctx := newStreamContext(p.ctx, p.logger, conn)
	defer ctx.Close()

	p.eventStream.Send(ctx, EventStart{
		CreatedAt: time.Now(),
		ConnID:    ctx.connID,
		RemoteIP:  ctx.ClientIP(),
	})
	ctx.logger.Info("Stream has been started")

	defer func() {
		p.eventStream.Send(ctx, EventFinish{
			CreatedAt: time.Now(),
			ConnID:    ctx.connID,
		})
		ctx.logger.Info("Stream has been finished")
	}()
}

func (p *Proxy) Serve(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("cannot accept a new connection: %w", err)
		}

		if addr := conn.RemoteAddr().(*net.TCPAddr).IP; p.ipBlocklist.Contains(addr) {
			conn.Close()
			p.eventStream.Send(p.ctx, EventIPBlocklisted{
				CreatedAt: time.Now(),
				RemoteIP:  addr,
			})

			continue
		}

		err = p.workerPool.Invoke(conn)

		switch {
		case err == nil:
		case errors.Is(err, ants.ErrPoolClosed):
			return nil
		case errors.Is(err, ants.ErrPoolOverload):
			p.eventStream.Send(p.ctx, EventConcurrencyLimited{})
		}
	}
}

func (p *Proxy) Shutdown() {
	p.ctxCancel()
	p.streamWaitGroup.Wait()
	p.workerPool.Release()
}

type antsLogger struct{}

func (a antsLogger) Printf(msg string, args ...interface{}) {}

func NewProxy(opts ProxyOpts) (*Proxy, error) {
	switch {
	case opts.Network == nil:
		return nil, ErrNetworkIsNotDefined
	case opts.AntiReplayCache == nil:
		return nil, ErrAntiReplayCacheIsNotDefined
	case opts.IPBlocklist == nil:
		return nil, ErrIPBlocklistIsNotDefined
	case opts.EventStream == nil:
		return nil, ErrEventStreamIsNotDefined
	case opts.Logger == nil:
		return nil, ErrLoggerIsNotDefined
	case !opts.Secret.Valid():
		return nil, ErrSecretInvalid
	}

	concurrency := opts.Concurrency
	if concurrency == 0 {
		concurrency = DefaultConcurrency
	}

	ctx, cancel := context.WithCancel(context.Background())
	proxy := &Proxy{
		ctx:             ctx,
		ctxCancel:       cancel,
		secret:          opts.Secret,
		network:         opts.Network,
		antiReplayCache: opts.AntiReplayCache,
		ipBlocklist:     opts.IPBlocklist,
		eventStream:     opts.EventStream,
		logger:          opts.Logger.Named("proxy"),
	}

	pool, err := ants.NewPoolWithFunc(int(concurrency), func(arg interface{}) {
		proxy.ServeConn(arg.(net.Conn))
	}, ants.WithLogger(antsLogger{}))
	if err != nil {
		return nil, fmt.Errorf("cannot initialize a pool: %w", err)
	}

	proxy.workerPool = pool

	return proxy, nil
}
