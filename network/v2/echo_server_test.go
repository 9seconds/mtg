package network_test

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type EchoServer struct {
	wg        sync.WaitGroup
	ctx       context.Context
	ctxCancel context.CancelFunc
	listener  net.Listener
}

func (e *EchoServer) Run() {
	e.wg.Go(func() {
		<-e.ctx.Done()
		e.listener.Close()
	})

	e.wg.Go(func() {
		for {
			conn, err := e.listener.Accept()
			if err != nil {
				return
			}

			e.wg.Go(func() {
				<-e.ctx.Done()
				conn.Close()
			})
			e.wg.Go(func() {
				e.process(conn)
			})
		}
	})
}

func (e *EchoServer) Stop() {
	e.ctxCancel()
	e.wg.Wait()
}

func (e *EchoServer) Addr() string {
	return e.listener.Addr().String()
}

func (e *EchoServer) process(conn io.ReadWriter) {
	buf := [4096]byte{}

	for {
		select {
		case <-e.ctx.Done():
			return
		default:
		}

		n, err := conn.Read(buf[:])
		if err != nil {
			return
		}

		select {
		case <-e.ctx.Done():
			return
		default:
		}

		if _, err = conn.Write(buf[:n]); err != nil {
			return
		}
	}
}

type EchoServerTestSuite struct {
	suite.Suite

	echoServer *EchoServer
}

func (suite *EchoServerTestSuite) SetupSuite() {
	ctx, cancel := context.WithCancel(context.Background())

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(suite.T(), err)

	suite.echoServer = &EchoServer{
		ctx:       ctx,
		ctxCancel: cancel,
		listener:  listener,
	}
	suite.echoServer.Run()
}

func (suite *EchoServerTestSuite) TearDownSuite() {
	suite.echoServer.Stop()
}

func (suite *EchoServerTestSuite) EchoServerAddr() string {
	return suite.echoServer.Addr()
}
