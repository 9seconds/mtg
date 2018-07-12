package rwc

import (
	"net"

	"github.com/juju/errors"
	"go.uber.org/zap"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/mtproto/rpc"
	"github.com/9seconds/mtg/multiplexer"
	"github.com/9seconds/mtg/wrappers"
)

type Multiplexer struct {
	socketID    string
	connID      multiplexer.ConnectionID
	connIDBytes []byte
	readChan    <-chan rpc.ProxyResponse
	logger      *zap.SugaredLogger
	opts        *mtproto.ConnectionOpts
}

func (m *Multiplexer) Read() ([]byte, error) {
	resp, err := multiplexer.Read(m.readChan)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read from multiplexer")
	}
	if resp.ResponseType() == rpc.ProxyResponseTypeCloseExt {
		return nil, errors.New("Telegram has closed connection")
	}

	return resp.Data(), nil
}

func (m *Multiplexer) Write(p []byte) (int, error) {
	multiplexer.Write(p, m.opts, m.connID)
	return len(p), nil
}

func (m *Multiplexer) Close() error {
	multiplexer.Deregister(m.connID)
	return nil
}

func (m *Multiplexer) Logger() *zap.SugaredLogger {
	return m.logger
}

func (m *Multiplexer) LocalAddr() *net.TCPAddr {
	return nil
}

func (m *Multiplexer) RemoteAddr() *net.TCPAddr {
	return nil
}

func (m *Multiplexer) SocketID() string {
	return m.socketID
}

func NewMultiplexer(opts *mtproto.ConnectionOpts, socketID string, connID []byte) (wrappers.PacketReadWriteCloser, error) {
	id, err := multiplexer.ToConnectionID(connID)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot initialize new pre-multiplexer")
	}

	readChan, err := multiplexer.Register(id)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot initialize new pre-multiplexer")
	}

	logger := zap.S().With(
		"socket_id", socketID,
		"purpose", "telegram",
	).Named("multiplexer")

	return &Multiplexer{
		connIDBytes: connID,
		connID:      id,
		logger:      logger,
		opts:        opts,
		readChan:    readChan,
		socketID:    socketID,
	}, nil
}
