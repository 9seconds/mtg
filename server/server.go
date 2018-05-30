package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/9seconds/mtg/obfuscated2"
	"github.com/juju/errors"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

type Server struct {
	ip     net.IP
	port   int
	secret []byte
	logger *zap.SugaredLogger
	lsock  net.Listener
	ctx    context.Context
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

	ctx, cancel := context.WithCancel(context.Background())
	socketID := s.makeSocketID()

	s.logger.Debugw("Client connected",
		"secret", s.secret,
		"addr", conn.RemoteAddr().String(),
		"socketid", socketID,
	)

	clientCipher, dc := s.getClientStream(conn, ctx, cancel, socketID)
	// if err != nil {
	// 	s.logger.Warnw("Cannot initialize client connection",
	// 		"secret", s.secret,
	// 		"addr", conn.RemoteAddr().String(),
	// 		"socketid", socketID,
	// 		"error", err,
	// 	)
	// 	return
	// }

	tgConn, tgCipher := s.getTelegramStream(dc, ctx, cancel, socketID)
	// if err != nil {
	// 	s.logger.Warnw("Cannot initialize Telegram connection",
	// 		"socketid", socketID,
	// 		"error", err,
	// 	)
	// 	return
	// }
	defer tgConn.Close()

	wait := &sync.WaitGroup{}
	wait.Add(2)
	go func() {
		buf := make([]byte, 128)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			decrypted := clientCipher.Decrypt(buf[:n])
			encrypted := tgCipher.Encrypt(decrypted)
			tgConn.Write(encrypted)
		}
	}()
	go func() {
		buf := make([]byte, 128)
		for {
			n, err := tgConn.Read(buf)
			if err != nil {
				return
			}
			decrypted := tgCipher.Decrypt(buf[:n])
			encrypted := clientCipher.Encrypt(decrypted)
			conn.Write(encrypted)
		}
	}()
	// go func() {
	// 	buf := make([]byte, 128)
	// 	for {
	// 		n, err := conn.
	// 	}
	// 	defer wait.Done()
	// 	io.Copy(tgConn, clientConn)
	// }()
	// go func() {
	// 	defer wait.Done()
	// 	io.Copy(clientConn, tgConn)
	// }()
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

func (s *Server) getClientStream(conn net.Conn, ctx context.Context, cancel context.CancelFunc, socketID string) (Cipher, int16) {
	frame, err := obfuscated2.ExtractFrame(conn)
	if err != nil {
		fmt.Println(err)
		// return nil, 0, errors.Annotate(err, "Cannot create client stream")
	}

	obfs2, dc, err := obfuscated2.ParseObfuscated2ClientFrame(s.secret, frame)
	if err != nil {
		fmt.Println(err)
		// return nil, 0, errors.Annotate(err, "Cannot create client stream")
	}

	return obfs2, dc

	// cipherConn := newCipherReadWriteCloser(conn, obfs2)
	// ctxConn := newCtxReadWriteCloser(cipherConn, ctx, cancel)
	// // logConn := newLogReadWriteCloser(ctxConn, s.logger, socketID, "client")

	// return ctxConn, dc, nil
}

func (s *Server) getTelegramStream(dc int16, ctx context.Context, cancel context.CancelFunc, socketID string) (io.ReadWriteCloser, Cipher) {
	socket, err := dialToTelegram(dc)
	if err != nil {
		fmt.Println(err)
		// return nil, errors.Annotate(err, "Cannot dial")
	}

	obfs2, frame := obfuscated2.MakeTelegramObfuscated2Frame()
	if n, err := socket.Write(frame); err != nil || n != len(frame) {
		fmt.Println(err)
		// return nil, errors.Annotate(err, "Cannot write hadnshake frame")
	}

	return socket, obfs2

	// 	cipherConn := newCipherReadWriteCloser(socket, obfs2)
	// 	ctxConn := newCtxReadWriteCloser(cipherConn, ctx, cancel)
	// 	logConn := newLogReadWriteCloser(ctxConn, s.logger, socketID, "telegram")

	// 	return logConn, nil
}

func NewServer(ip net.IP, port int, secret []byte, logger *zap.SugaredLogger) *Server {
	return &Server{
		ip:     ip,
		port:   port,
		secret: secret,
		ctx:    context.Background(),
		logger: logger,
	}
}
