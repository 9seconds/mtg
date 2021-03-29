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
	processMutex sync.Mutex
	eastBuffer   []byte
	westBuffer   []byte
	tickChannel  chan struct{}
	errorChannel chan error
	tickTimeout  time.Duration
}

func (r *Relay) Reset() {
	r.processMutex.Lock()
	defer r.processMutex.Unlock()

	if r.ctxCancel != nil {
		r.ctxCancel()
	}

	r.ctx = nil
	r.ctxCancel = nil
	r.logger = nil
}

func (r *Relay) Process(eastConn, westConn io.ReadWriteCloser) error {
	r.processMutex.Lock()
	defer r.processMutex.Unlock()

	eastConn = conn{
		ReadWriteCloser: eastConn,
		ctx:             r.ctx,
		tickChannel:     r.tickChannel,
	}
	westConn = conn{
		ReadWriteCloser: westConn,
		ctx:             r.ctx,
		tickChannel:     r.tickChannel,
	}

	wg := &sync.WaitGroup{}
	wg.Add(3) // nolint: gomnd

	go r.runObserver(eastConn, westConn, wg)

	go r.transmit(eastConn, westConn, r.westBuffer, "west", wg)

	r.transmit(westConn, eastConn, r.eastBuffer, "east", wg)

	wg.Wait()

	select {
	case err := <-r.errorChannel:
		return err
	default:
		return nil
	}
}

func (r *Relay) transmit(src io.ReadCloser, dst io.WriteCloser,
	buffer []byte, direction string, wg *sync.WaitGroup) {
	defer func() {
		src.Close()
		dst.Close()
		wg.Done()
		r.ctxCancel()
	}()

	if _, err := io.CopyBuffer(dst, src, buffer); err != nil {
		r.logger.Printf("error '%v' happened on direction %s", err, direction)

		select {
		case <-r.ctx.Done():
			err = r.ctx.Err()
		default:
		}

		select {
		case r.errorChannel <- err:
		default:
		}
	}
}

func (r *Relay) runObserver(one, another io.Closer, wg *sync.WaitGroup) {
	ticker := time.NewTicker(time.Second)

	defer func() {
		one.Close()
		another.Close()

		ticker.Stop()

		select {
		case <-ticker.C:
		default:
		}

		wg.Done()
	}()

	lastTickAt := time.Now()

	for {
		select {
		case <-r.ctx.Done():
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
