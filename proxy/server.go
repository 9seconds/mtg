package proxy

import (
	"context"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/9seconds/mtg/obfuscated2"
	"github.com/juju/errors"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

const bufferSize = 4096

type Server struct {
	ip           net.IP
	port         int
	secret       []byte
	logger       *zap.SugaredLogger
	lsock        net.Listener
	ctx          context.Context
	readTimeout  time.Duration
	writeTimeout time.Duration
	stats        *Stats
	ipv6         bool
}

func (s *Server) Serve() error {
	lsock, err := net.Listen("tcp", s.Addr())
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

	return nil
}

func (s *Server) Addr() string {
	return net.JoinHostPort(s.ip.String(), strconv.Itoa(s.port))
}

func (s *Server) accept(conn net.Conn) {
	defer conn.Close()
	defer s.stats.closeConnection()
	s.stats.newConnection()

	ctx, cancel := context.WithCancel(context.Background())
	socketID := s.makeSocketID()

	s.logger.Debugw("Client connected",
		"secret", s.secret,
		"addr", conn.RemoteAddr().String(),
		"socketid", socketID,
	)

	clientConn, dc, err := s.getClientStream(conn, ctx, cancel, socketID)
	if err != nil {
		s.logger.Warnw("Cannot initialize client connection",
			"secret", s.secret,
			"addr", conn.RemoteAddr().String(),
			"socketid", socketID,
			"error", err,
		)
		return
	}
	defer clientConn.Close()

	tgConn, err := s.getTelegramStream(dc, ctx, cancel, socketID)
	if err != nil {
		s.logger.Warnw("Cannot initialize Telegram connection",
			"socketid", socketID,
			"error", err,
		)
		return
	}
	defer tgConn.Close()

	wait := &sync.WaitGroup{}
	wait.Add(2)
	go s.pipe(wait, clientConn, tgConn)
	go s.pipe(wait, tgConn, clientConn)
	<-ctx.Done()
	wait.Wait()

	s.logger.Debugw("Client disconnected",
		"secret", s.secret,
		"addr", conn.RemoteAddr().String(),
		"socketid", socketID,
	)
}

func (s *Server) makeSocketID() string {
	return uuid.NewV4().String()
}

func (s *Server) getClientStream(conn net.Conn, ctx context.Context, cancel context.CancelFunc, socketID string) (io.ReadWriteCloser, int16, error) {
	wConn := newTimeoutReadWriteCloser(conn, s.readTimeout, s.writeTimeout)
	wConn = newTrafficReadWriteCloser(wConn, s.stats.addIncomingTraffic, s.stats.addOutgoingTraffic)
	frame, err := obfuscated2.ExtractFrame(wConn)
	if err != nil {
		return nil, 0, errors.Annotate(err, "Cannot create client stream")
	}

	obfs2, dc, err := obfuscated2.ParseObfuscated2ClientFrame(s.secret, frame)
	if err != nil {
		return nil, 0, errors.Annotate(err, "Cannot create client stream")
	}

	wConn = newLogReadWriteCloser(wConn, s.logger, socketID, "client")
	wConn = newCipherReadWriteCloser(wConn, obfs2)
	wConn = newCtxReadWriteCloser(wConn, ctx, cancel)

	return wConn, dc, nil
}

func (s *Server) getTelegramStream(dc int16, ctx context.Context, cancel context.CancelFunc, socketID string) (io.ReadWriteCloser, error) {
	socket, err := dialToTelegram(s.ipv6, dc, s.readTimeout)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot dial")
	}
	wConn := newTimeoutReadWriteCloser(socket, s.readTimeout, s.writeTimeout)
	wConn = newTrafficReadWriteCloser(wConn, s.stats.addIncomingTraffic, s.stats.addOutgoingTraffic)

	obfs2, frame := obfuscated2.MakeTelegramObfuscated2Frame()
	if n, err := socket.Write(frame); err != nil || n != len(frame) {
		return nil, errors.Annotate(err, "Cannot write hadnshake frame")
	}

	wConn = newLogReadWriteCloser(wConn, s.logger, socketID, "telegram")
	wConn = newCipherReadWriteCloser(wConn, obfs2)
	wConn = newCtxReadWriteCloser(wConn, ctx, cancel)

	return wConn, nil
}

func (s *Server) pipe(wait *sync.WaitGroup, reader io.Reader, writer io.Writer) {
	defer wait.Done()
	io.Copy(writer, reader)
}

func NewServer(ip net.IP, port int, secret []byte, logger *zap.SugaredLogger,
	readTimeout, writeTimeout time.Duration, ipv6 bool, stat *Stats) *Server {
	return &Server{
		ip:           ip,
		port:         port,
		secret:       secret,
		ctx:          context.Background(),
		logger:       logger,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		stats:        stat,
		ipv6:         ipv6,
	}
}
