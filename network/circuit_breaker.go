package network

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
)

const (
	circuitBreakerStateClosed uint32 = iota
	circuitBreakerStateHalfOpened
	circuitBreakerStateOpened
)

type circuitBreakerDialer struct {
	Dialer

	stateMutexChan chan bool

	halfOpenTimer        *time.Timer
	failuresCleanupTimer *time.Timer

	state            uint32
	halfOpenAttempts uint32
	failuresCount    uint32

	openThreshold        uint32
	halfOpenTimeout      time.Duration
	resetFailuresTimeout time.Duration
}

func (c *circuitBreakerDialer) Dial(network, address string) (essentials.Conn, error) {
	return c.DialContext(context.Background(), network, address)
}

func (c *circuitBreakerDialer) DialContext(ctx context.Context,
	network, address string,
) (essentials.Conn, error) {
	switch atomic.LoadUint32(&c.state) {
	case circuitBreakerStateClosed:
		return c.doClosed(ctx, network, address)
	case circuitBreakerStateHalfOpened:
		return c.doHalfOpened(ctx, network, address)
	default:
		return nil, ErrCircuitBreakerOpened
	}
}

func (c *circuitBreakerDialer) doClosed(ctx context.Context,
	network, address string,
) (essentials.Conn, error) {
	conn, err := c.Dialer.DialContext(ctx, network, address)

	select {
	case <-ctx.Done():
		if conn != nil {
			conn.Close()
		}

		return nil, ctx.Err() //nolint: wrapcheck
	case c.stateMutexChan <- true:
		defer func() {
			<-c.stateMutexChan
		}()
	}

	if err == nil {
		c.switchState(circuitBreakerStateClosed)

		return conn, nil
	}

	c.failuresCount++

	if c.state == circuitBreakerStateClosed && c.failuresCount >= c.openThreshold {
		c.switchState(circuitBreakerStateOpened)
	}

	return conn, err //nolint: wrapcheck
}

func (c *circuitBreakerDialer) doHalfOpened(ctx context.Context,
	network, address string,
) (essentials.Conn, error) {
	if !atomic.CompareAndSwapUint32(&c.halfOpenAttempts, 0, 1) {
		return nil, ErrCircuitBreakerOpened
	}

	conn, err := c.Dialer.DialContext(ctx, network, address)

	select {
	case <-ctx.Done():
		if conn != nil {
			conn.Close()
		}

		return nil, ctx.Err() //nolint: wrapcheck
	case c.stateMutexChan <- true:
		defer func() {
			<-c.stateMutexChan
		}()
	}

	if c.state != circuitBreakerStateHalfOpened {
		return conn, err //nolint: wrapcheck
	}

	if err == nil {
		c.switchState(circuitBreakerStateClosed)
	} else {
		c.switchState(circuitBreakerStateOpened)
	}

	return conn, err //nolint: wrapcheck
}

func (c *circuitBreakerDialer) switchState(state uint32) {
	switch state {
	case circuitBreakerStateClosed:
		c.stopTimer(&c.halfOpenTimer)
		c.ensureTimer(&c.failuresCleanupTimer, c.resetFailuresTimeout, c.resetFailures)
	case circuitBreakerStateHalfOpened:
		c.stopTimer(&c.failuresCleanupTimer)
		c.stopTimer(&c.halfOpenTimer)
	case circuitBreakerStateOpened:
		c.stopTimer(&c.failuresCleanupTimer)
		c.ensureTimer(&c.halfOpenTimer, c.halfOpenTimeout, c.tryHalfOpen)
	}

	c.failuresCount = 0
	atomic.StoreUint32(&c.halfOpenAttempts, 0)
	atomic.StoreUint32(&c.state, state)
}

func (c *circuitBreakerDialer) resetFailures() {
	c.stateMutexChan <- true

	defer func() {
		<-c.stateMutexChan
	}()

	c.stopTimer(&c.failuresCleanupTimer)

	if c.state == circuitBreakerStateClosed {
		c.switchState(circuitBreakerStateClosed)
	}
}

func (c *circuitBreakerDialer) tryHalfOpen() {
	c.stateMutexChan <- true

	defer func() {
		<-c.stateMutexChan
	}()

	if c.state == circuitBreakerStateOpened {
		c.switchState(circuitBreakerStateHalfOpened)
	}
}

func (c *circuitBreakerDialer) stopTimer(timerRef **time.Timer) {
	timer := *timerRef
	if timer == nil {
		return
	}

	timer.Stop()

	select {
	case <-timer.C:
	default:
	}

	*timerRef = nil
}

func (c *circuitBreakerDialer) ensureTimer(timerRef **time.Timer,
	timeout time.Duration, callback func(),
) {
	if *timerRef == nil {
		*timerRef = time.AfterFunc(timeout, callback)
	}
}

func newCircuitBreakerDialer(baseDialer Dialer,
	openThreshold uint32, halfOpenTimeout, resetFailuresTimeout time.Duration,
) Dialer {
	cb := &circuitBreakerDialer{
		Dialer:               baseDialer,
		stateMutexChan:       make(chan bool, 1),
		openThreshold:        openThreshold,
		halfOpenTimeout:      halfOpenTimeout,
		resetFailuresTimeout: resetFailuresTimeout,
	}

	cb.stateMutexChan <- true // to convince race detector we are good
	cb.switchState(circuitBreakerStateClosed)
	<-cb.stateMutexChan

	return cb
}
