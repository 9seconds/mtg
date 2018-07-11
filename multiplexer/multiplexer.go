package multiplexer

import (
	"sync"

	"github.com/juju/errors"
	"go.uber.org/zap"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/mtproto/rpc"
	"github.com/9seconds/mtg/telegram"
	"github.com/9seconds/mtg/wrappers"
)

type ConnectionID uint64

const (
	writeChanLength = 5000
	readChanLength  = 5
)

var instance *Multiplexer

type Multiplexer struct {
	pools     map[mtproto.ConnectionProtocol]map[int16]*connectionPool
	logger    *zap.SugaredLogger
	dialer    telegram.TelegramMiddleDialer
	writeChan chan *Request
	readChans *sync.Map
}

func (m *Multiplexer) register(id ConnectionID) (<-chan rpc.ProxyResponse, error) {
	readChan, ok := m.readChans.LoadOrStore(id, make(chan rpc.ProxyResponse, readChanLength))
	if ok {
		return nil, errors.Errorf("Such connection ID was alread registered: %d", int64(id))
	}
	m.logger.Debugw("Register connection", "id", id)

	return readChan.(chan rpc.ProxyResponse), nil
}

func (m *Multiplexer) deregister(id ConnectionID) {
	m.logger.Debugw("Deregister connection", "id", id)
	m.readChans.Delete(id)
}

func (m *Multiplexer) getConnection(req *Request) (wrappers.PacketReadWriteCloser, error) {
	pool, ok := m.pools[req.protocol][req.dc]
	if !ok {
		pool = newConnectionPool(m.logger, m.dialer, req.protocol, req.dc)
		m.pools[req.protocol][req.dc] = pool
	}

	conn, isNew, err := pool.get()
	if err != nil {
		return nil, err
	}
	if isNew {
		go m.readLoop(conn)
	}

	return conn, nil
}

func (m *Multiplexer) returnConnection(req *Request, conn wrappers.PacketReadWriteCloser) {
	m.pools[req.protocol][req.dc].put(conn)
}

func (m *Multiplexer) readLoop(conn wrappers.PacketReadWriteCloser) {
	for {
		data, err := conn.Read()
		if err != nil {
			m.logger.Debugw("Cannot read from Telegram", "error", err)
			conn.Close()
			return
		}

		resp, err := rpc.GetProxyResponse(data)
		if err != nil {
			m.logger.Debugw("Cannot read correct response", "error", err)
			conn.Close()
			return
		}
		connID, _ := ToConnectionID(resp.ConnectionID())
		readChanRaw, ok := m.readChans.Load(connID)
		if !ok {
			m.logger.Warnw("Cannot find channel to send response",
				"connection_d", resp.ConnectionID(),
				"numeric_connection_id", uint64(connID),
			)
			return
		}
		readChanRaw.(chan<- rpc.ProxyResponse) <- resp

		if resp.ResponseType() == rpc.ProxyResponseTypeCloseExt {
			m.logger.Infow("Telegram has asked to close the connection")
			conn.Close()
			return
		}
	}
}

func (m *Multiplexer) writeLoop() {
	for req := range m.writeChan {
		conn, err := m.getConnection(req)
		readChanRaw, ok := m.readChans.Load(req.connID)
		if !ok {
			panic("Unregistered request")
		}
		readChan := readChanRaw.(chan rpc.ProxyResponse)

		if err != nil {
			m.logger.Debugw("Cannot get connection to Telegram", "error", err)
			close(readChan)
			continue
		}

		go func(req *Request, conn wrappers.PacketReadWriteCloser, readChan chan rpc.ProxyResponse) {
			if _, err := conn.Write(req.data); err != nil {
				m.logger.Debugw("Cannot write data", "error", err)
				close(readChan)
				conn.Close()
			} else {
				m.returnConnection(req, conn)
			}
		}(req, conn, readChan)
	}
}

func InitMultiplexer(dialer telegram.TelegramMiddleDialer) {
	instance = &Multiplexer{
		logger: zap.S().Named("multiplexer"),
		dialer: dialer,
		pools: map[mtproto.ConnectionProtocol]map[int16]*connectionPool{
			mtproto.ConnectionProtocolIPv4: map[int16]*connectionPool{},
			mtproto.ConnectionProtocolIPv6: map[int16]*connectionPool{},
		},
		writeChan: make(chan *Request, writeChanLength),
		readChans: &sync.Map{},
	}
	go instance.writeLoop()
}
