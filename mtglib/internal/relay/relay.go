package relay

import (
	"context"
	"io"
	"sync"
	"time"
)

type Relay struct {
	ctx          context.Context
	ctxCancel    context.CancelFunc
	logger       Logger
	eastBuffer   []byte
	westBuffer   []byte
	tickChannel  chan struct{}
	errorChannel chan error
	tickTimeout  time.Duration
}

func (r *Relay) Process(eastConn, westConn io.ReadWriteCloser) error {
	eastConn = conn{
		ReadWriteCloser: eastConn,
		relay:           r,
	}
	westConn = conn{
		ReadWriteCloser: westConn,
		relay:           r,
	}

	defer func() {
		r.ctxCancel()
		eastConn.Close()
		westConn.Close()
	}()

	go r.runObserver()

	wg := &sync.WaitGroup{}
	wg.Add(2) // nolint: gomnd

	go r.transmit(eastConn, westConn, r.westBuffer, "west", wg)

	r.transmit(westConn, eastConn, r.eastBuffer, "east", wg)

	wg.Wait()

	return <-r.errorChannel
}

func (r *Relay) transmit(src io.ReadCloser, dst io.WriteCloser,
	buffer []byte, direction string, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		src.Close()
		dst.Close()
	}()

	if _, err := io.CopyBuffer(dst, src, buffer); err != nil {
		r.logger.Printf("error '%v' happened on direction %s", err, direction)

		select {
		case <-r.ctx.Done():
		case r.errorChannel <- err:
		default:
		}
	}
}

func (r *Relay) runObserver() {
	ticker := time.NewTicker(time.Second)

	defer func() {
		ticker.Stop()

		select {
		case <-ticker.C:
		default:
		}
	}()

	lastTickAt := time.Now()
	ctx := r.ctx

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.tickChannel:
			lastTickAt = time.Now()
		case <-ticker.C:
			if time.Since(lastTickAt) > r.tickTimeout {
				r.logger.Printf("exit due to a timeout")
				r.ctxCancel()

				return
			}
		}
	}
}