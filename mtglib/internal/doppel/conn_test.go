package doppel

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ConnMock struct {
	testlib.EssentialsConnMock

	mu          sync.Mutex
	writeBuffer bytes.Buffer
}

func (m *ConnMock) Write(p []byte) (int, error) {
	args := m.Called(p)
	if err := args.Error(1); err != nil {
		return args.Int(0), err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.writeBuffer.Write(p)
}

func (m *ConnMock) Written() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	return bytes.Clone(m.writeBuffer.Bytes())
}

type ConnTestSuite struct {
	suite.Suite

	connMock  *ConnMock
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func (suite *ConnTestSuite) SetupTest() {
	ctx, cancel := context.WithCancel(context.Background())
	suite.ctx = ctx
	suite.ctxCancel = cancel
	suite.connMock = &ConnMock{}
}

func (suite *ConnTestSuite) TearDownTest() {
	suite.ctxCancel()
	suite.connMock.AssertExpectations(suite.T())
}

func (suite *ConnTestSuite) makeConn() Conn {
	return NewConn(suite.ctx, suite.connMock, &Stats{
		k:      2.0,
		lambda: 0.01,
	})
}

func (suite *ConnTestSuite) TestWriteBuffersData() {
	suite.connMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(0, nil).
		Maybe()

	c := suite.makeConn()
	defer c.Stop()

	n, err := c.Write([]byte{1, 2, 3})
	suite.NoError(err)
	suite.Equal(3, n)
}

func (suite *ConnTestSuite) TestWriteOutputsTLSRecords() {
	suite.connMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(0, nil).
		Maybe()

	c := suite.makeConn()

	payload := []byte("hello doppelganger")
	_, err := c.Write(payload)
	suite.NoError(err)

	suite.Eventually(func() bool {
		return len(suite.connMock.Written()) > 0
	}, 2*time.Second, time.Millisecond)

	c.Stop()

	assembled := &bytes.Buffer{}
	reader := bytes.NewReader(suite.connMock.Written())

	for {
		header := make([]byte, tls.SizeHeader)
		if _, err := io.ReadFull(reader, header); err != nil {
			break
		}

		suite.Equal(byte(tls.TypeApplicationData), header[0])
		suite.Equal(tls.TLSVersion[:], header[tls.SizeRecordType:tls.SizeRecordType+tls.SizeVersion])

		length := binary.BigEndian.Uint16(header[tls.SizeRecordType+tls.SizeVersion:])
		suite.Greater(length, uint16(0))

		rec := make([]byte, length)
		_, err := io.ReadFull(reader, rec)
		suite.NoError(err)

		assembled.Write(rec)
	}

	suite.Equal(payload, assembled.Bytes())
}

func (suite *ConnTestSuite) TestWriteReturnsErrorAfterStop() {
	suite.connMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(0, nil).
		Maybe()

	c := suite.makeConn()
	c.Stop()

	time.Sleep(10 * time.Millisecond)

	_, err := c.Write([]byte{1})
	suite.Error(err)
}

func (suite *ConnTestSuite) TestStopOnUnderlyingWriteError() {
	suite.connMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(0, errors.New("connection reset")).
		Maybe()

	c := suite.makeConn()

	_, _ = c.Write([]byte("data"))

	suite.Eventually(func() bool {
		_, err := c.Write([]byte{1})
		return err != nil
	}, 2*time.Second, time.Millisecond)
}

func (suite *ConnTestSuite) TestSyncWriteDataSent() {
	suite.connMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(0, nil).
		Maybe()

	c := suite.makeConn()
	defer c.Stop()

	payload := []byte("sync hello")
	n, err := c.SyncWrite(payload)
	suite.NoError(err)
	suite.Equal(len(payload), n)

	// SyncWrite returns only after data is flushed to the wire.
	assembled := &bytes.Buffer{}
	reader := bytes.NewReader(suite.connMock.Written())

	for {
		header := make([]byte, tls.SizeHeader)
		if _, err := io.ReadFull(reader, header); err != nil {
			break
		}

		suite.Equal(byte(tls.TypeApplicationData), header[0])

		length := binary.BigEndian.Uint16(header[tls.SizeRecordType+tls.SizeVersion:])
		rec := make([]byte, length)
		_, err := io.ReadFull(reader, rec)
		suite.NoError(err)

		assembled.Write(rec)
	}

	suite.Equal(payload, assembled.Bytes())
}

func (suite *ConnTestSuite) TestSyncWriteDrainsBufferFirst() {
	suite.connMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(0, nil).
		Maybe()

	c := suite.makeConn()
	defer c.Stop()

	// Buffer some data via async Write.
	_, err := c.Write([]byte("first"))
	suite.NoError(err)

	// SyncWrite must drain "first" before sending "second".
	n, err := c.SyncWrite([]byte("second"))
	suite.NoError(err)
	suite.Equal(6, n)

	// All data should be on the wire now.
	assembled := &bytes.Buffer{}
	reader := bytes.NewReader(suite.connMock.Written())

	for {
		header := make([]byte, tls.SizeHeader)
		if _, err := io.ReadFull(reader, header); err != nil {
			break
		}

		length := binary.BigEndian.Uint16(header[tls.SizeRecordType+tls.SizeVersion:])
		rec := make([]byte, length)
		_, err := io.ReadFull(reader, rec)
		suite.NoError(err)

		assembled.Write(rec)
	}

	suite.Equal([]byte("firstsecond"), assembled.Bytes())
}

func (suite *ConnTestSuite) TestSyncWriteBlocksAsyncWrite() {
	suite.connMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(0, nil).
		Maybe()

	c := suite.makeConn()
	defer c.Stop()

	// Start SyncWrite — it holds exclusive lock.
	syncDone := make(chan struct{})

	go func() {
		defer close(syncDone)
		c.SyncWrite([]byte("exclusive")) //nolint: errcheck
	}()

	// Give SyncWrite time to acquire the lock.
	time.Sleep(10 * time.Millisecond)

	// Async Write should block until SyncWrite completes.
	writeDone := make(chan struct{})

	go func() {
		defer close(writeDone)
		c.Write([]byte("blocked")) //nolint: errcheck
	}()

	// SyncWrite should finish first.
	<-syncDone

	select {
	case <-writeDone:
		// Write completed after SyncWrite — correct.
	case <-time.After(2 * time.Second):
		suite.Fail("async Write did not unblock after SyncWrite completed")
	}
}

func (suite *ConnTestSuite) TestSyncWriteReturnsErrorAfterStop() {
	suite.connMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(0, nil).
		Maybe()

	c := suite.makeConn()
	c.Stop()

	time.Sleep(10 * time.Millisecond)

	_, err := c.SyncWrite([]byte("too late"))
	suite.Error(err)
}

func TestConn(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConnTestSuite{})
}
