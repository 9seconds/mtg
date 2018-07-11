package multiplexer

import (
	"container/list"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/telegram"
	"github.com/9seconds/mtg/wrappers"
)

const connectionPoolCleanupEvery = 10 * time.Second

var pools map[mtproto.ConnectionProtocol]map[int16]*connectionPool

type connectionPool struct {
	queue    *list.List
	pressure bool
	dc       int16
	protocol mtproto.ConnectionProtocol
	tg       telegram.TelegramMiddleDialer
	lock     *sync.Mutex
	logger   *zap.SugaredLogger
}

func (c *connectionPool) get() (wrappers.PacketReadWriteCloser, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.queue.Len() > 0 {
		return c.queue.Remove(c.queue.Front()).(wrappers.PacketReadWriteCloser), nil
	}

	c.pressure = true
	c.logger.Debugw("Cannot find out free connection, create new one")

	return c.tg.Dial(c.dc, c.protocol)
}

func (c *connectionPool) put(connection wrappers.PacketReadWriteCloser) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.queue.PushBack(connection)
}

func (c *connectionPool) autoCleanup() {
	for range time.Tick(connectionPoolCleanupEvery) {
		c.logger.Debugw("Start cleanup")

		c.lock.Lock()
		if !c.pressure && c.queue.Len() > 0 {
			data := c.queue.Remove(c.queue.Front()).(wrappers.Closer)
			data.Close()
			c.logger.Debugw("Removed Telegram connection", "socketid", data.SocketID())
		} else {
			c.logger.Debugw("Nothing to cleanup yet")
		}
		c.pressure = false
		c.lock.Unlock()

		c.logger.Debugw("Finish cleanup")
	}
}

func newConnectionPool(logger *zap.SugaredLogger, dc int16, proto mtproto.ConnectionProtocol, dialer telegram.TelegramMiddleDialer) *connectionPool {
	pool := &connectionPool{
		dc:       dc,
		lock:     &sync.Mutex{},
		logger:   logger.Named("connection-pool").With("dc", dc),
		protocol: proto,
		queue:    list.New(),
		tg:       dialer,
	}
	go pool.autoCleanup()

	return pool
}

func getConnection(logger *zap.SugaredLogger, dc int16, proto mtproto.ConnectionProtocol, dialer telegram.TelegramMiddleDialer) (wrappers.PacketReadWriteCloser, error) {
	pool, ok := pools[proto][dc]
	if !ok {
		pool = newConnectionPool(logger, dc, proto, dialer)
		pools[proto][dc] = pool
	}

	return pool.get()
}

func returnConnection(dc int16, proto mtproto.ConnectionProtocol, conn wrappers.PacketReadWriteCloser) {
	pools[proto][dc].put(conn)
}

func init() {
	pools = map[mtproto.ConnectionProtocol]map[int16]*connectionPool{
		mtproto.ConnectionProtocolIPv4: map[int16]*connectionPool{},
		mtproto.ConnectionProtocolIPv6: map[int16]*connectionPool{},
	}
}
