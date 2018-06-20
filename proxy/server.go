package proxy

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/juju/errors"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"

	"github.com/9seconds/mtg/client"
	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/telegram"
	"github.com/9seconds/mtg/wrappers"
)

// Server is an insgtance of MTPROTO proxy.
type Server struct {
	conf       *config.Config
	logger     *zap.SugaredLogger
	stats      *Stats
	tg         telegram.Telegram
	clientInit client.Init
}

// Serve does MTPROTO proxying.
func (s *Server) Serve() error {
	lsock, err := net.Listen("tcp", s.conf.BindAddr())
	if err != nil {
		return errors.Annotate(err, "Cannot create listen socket")
	}

	for {
		if conn, err := lsock.Accept(); err != nil {
			s.logger.Warn("Cannot allocate incoming connection", "error", err)
		} else {
			go s.accept(conn)
		}
	}
}

func (s *Server) accept(conn net.Conn) {
	defer func() {
		s.stats.closeConnection()
		conn.Close() // nolint: errcheck

		if r := recover(); r != nil {
			s.logger.Errorw("Crash of accept handler", "error", r)
		}
	}()

	s.stats.newConnection()
	ctx, cancel := context.WithCancel(context.Background())
	socketID := uuid.NewV4().String()

	s.logger.Debugw("Client connected",
		"addr", conn.RemoteAddr().String(),
		"socketid", socketID,
	)

	dc, clientConn, err := s.getClientStream(ctx, cancel, conn, socketID)
	if err != nil {
		s.logger.Warnw("Cannot initialize client connection",
			"addr", conn.RemoteAddr().String(),
			"socketid", socketID,
			"error", err,
		)
		return
	}
	defer clientConn.Close() // nolint: errcheck

	tgConn, err := s.getTelegramStream(ctx, cancel, dc, socketID)
	if err != nil {
		s.logger.Warnw("Cannot initialize Telegram connection",
			"socketid", socketID,
			"error", err,
		)
		return
	}
	defer tgConn.Close() // nolint: errcheck

	wait := &sync.WaitGroup{}
	wait.Add(2)

	go s.pipe(clientConn, tgConn, wait)
	go s.pipe(tgConn, clientConn, wait)

	<-ctx.Done()
	wait.Wait()

	s.logger.Debugw("Client disconnected",
		"addr", conn.RemoteAddr().String(),
		"socketid", socketID,
	)
}

func (s *Server) getClientStream(ctx context.Context, cancel context.CancelFunc, conn net.Conn, socketID string) (int16, io.ReadWriteCloser, error) {
	dc, socket, err := s.clientInit(conn, s.conf)
	if err != nil {
		return 0, nil, errors.Annotate(err, "Cannot init client connection")
	}

	socket = wrappers.NewTrafficRWC(socket, s.stats.addIncomingTraffic, s.stats.addOutgoingTraffic)
	socket = wrappers.NewLogRWC(socket, s.logger, socketID, "client")
	socket = wrappers.NewCtxRWC(ctx, cancel, socket)

	return dc, socket, nil
}

func (s *Server) getTelegramStream(ctx context.Context, cancel context.CancelFunc, dc int16, socketID string) (io.ReadWriteCloser, error) {
	conn, err := s.tg.Dial(dc)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot connect to Telegram")
	}

	conn = wrappers.NewTrafficRWC(conn, s.stats.addIncomingTraffic, s.stats.addOutgoingTraffic)
	conn, err = s.tg.Init(conn)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot handshake Telegram")
	}

	conn = wrappers.NewLogRWC(conn, s.logger, socketID, "telegram")
	conn = wrappers.NewCtxRWC(ctx, cancel, conn)

	return conn, nil
}

func (s *Server) pipe(dst io.Writer, src io.Reader, wait *sync.WaitGroup) {
	defer wait.Done()

	buf := copyPool.Get().(*[]byte)
	defer copyPool.Put(buf)

	io.CopyBuffer(dst, src, *buf) // nolint: errcheck
}

// NewServer creates new instance of MTPROTO proxy.
func NewServer(conf *config.Config, logger *zap.SugaredLogger, stat *Stats) *Server {
	return &Server{
		conf:       conf,
		logger:     logger,
		stats:      stat,
		tg:         telegram.NewDirectTelegram(conf),
		clientInit: client.DirectInit,
	}
}
