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

	wg := &sync.WaitGroup{}
	wg.Add(3) // nolint: gomnd

	go r.runObserver(r.ctx, wg)

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

func (r *Relay) runObserver(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(time.Second)

	defer func() {
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
