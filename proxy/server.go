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
	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/wrappers"
)

// Server is an insgtance of MTPROTO proxy.
type Server struct {
	conf   *config.Config
	logger *zap.SugaredLogger
	stats  *Stats
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

	clientConn, dc, err := s.getClientStream(ctx, cancel, conn, socketID)
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
	go func() {
		defer wait.Done()
		io.Copy(clientConn, tgConn) // nolint: errcheck
	}()
	go func() {
		defer wait.Done()
		io.Copy(tgConn, clientConn) // nolint: errcheck
	}()
	<-ctx.Done()
	wait.Wait()

	s.logger.Debugw("Client disconnected",
		"addr", conn.RemoteAddr().String(),
		"socketid", socketID,
	)
}

func (s *Server) getClientStream(ctx context.Context, cancel context.CancelFunc, conn net.Conn, socketID string) (io.ReadWriteCloser, int16, error) {
	wConn := wrappers.NewTimeoutRWC(conn, s.conf.TimeoutRead, s.conf.TimeoutWrite)
	wConn = wrappers.NewTrafficRWC(wConn, s.stats.addIncomingTraffic, s.stats.addOutgoingTraffic)
	frame, err := obfuscated2.ExtractFrame(wConn)
	if err != nil {
		return nil, 0, errors.Annotate(err, "Cannot create client stream")
	}

	obfs2, dc, err := obfuscated2.ParseObfuscated2ClientFrame(s.conf.Secret, frame)
	if err != nil {
		return nil, 0, errors.Annotate(err, "Cannot create client stream")
	}

	wConn = wrappers.NewLogRWC(wConn, s.logger, socketID, "client")
	wConn = wrappers.NewStreamCipherRWC(wConn, obfs2.Encryptor, obfs2.Decryptor)
	wConn = wrappers.NewCtxRWC(ctx, cancel, wConn)

	return wConn, dc, nil
}

func (s *Server) getTelegramStream(ctx context.Context, cancel context.CancelFunc, dc int16, socketID string) (io.ReadWriteCloser, error) {
	socket, err := dialToTelegram(dc, s.conf.TimeoutRead)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot dial")
	}
	wConn := wrappers.NewTimeoutRWC(socket, s.conf.TimeoutRead, s.conf.TimeoutWrite)
	wConn = wrappers.NewTrafficRWC(wConn, s.stats.addIncomingTraffic, s.stats.addOutgoingTraffic)

	obfs2, frame := obfuscated2.MakeTelegramObfuscated2Frame()
	if n, err := socket.Write(frame); err != nil || n != len(frame) {
		return nil, errors.Annotate(err, "Cannot write hadnshake frame")
	}

	wConn = wrappers.NewLogRWC(wConn, s.logger, socketID, "telegram")
	wConn = wrappers.NewStreamCipherRWC(wConn, obfs2.Encryptor, obfs2.Decryptor)
	wConn = wrappers.NewCtxRWC(ctx, cancel, wConn)

	return wConn, nil
}

// NewServer creates new instance of MTPROTO proxy.
func NewServer(conf *config.Config, logger *zap.SugaredLogger, stat *Stats) *Server {
	return &Server{
		conf:   conf,
		logger: logger,
		stats:  stat,
	}
}
