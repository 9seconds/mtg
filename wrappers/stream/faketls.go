package stream

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
)

var (
	errFakeTLSTimeout  = errors.New("timeout")
	fakeTLSWritePrefix = []byte{0x17, 0x03, 0x03}
)

const (
	faketlsMaxChunkSize              = 16384 + 24
	faketlsRecordTypeApplicationData = 0x17
	faketlsRecordTypeCCS             = 0x14
)

type wrapperFakeTLS struct {
	bufferedReader

	parent conntypes.StreamReadWriteCloser
}

func (w *wrapperFakeTLS) Write(p []byte) (int, error) {
	return w.write(p, func(b []byte) (int, error) {
		return w.parent.Write(b)
	})
}

func (w *wrapperFakeTLS) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	startTime := time.Now()

	return w.write(p, func(b []byte) (int, error) {
		elapsed := time.Since(startTime)
		if elapsed > timeout {
			return w.parent.WriteTimeout(b, timeout-elapsed)
		}
		return 0, errFakeTLSTimeout
	})
}

func (w *wrapperFakeTLS) write(p []byte, writeFunc func([]byte) (int, error)) (int, error) {
	sum := 0
	size := [2]byte{}

	for len(p) > 0 {
		chunkSize := faketlsMaxChunkSize
		if chunkSize > len(p) {
			chunkSize = len(p)
		}

		if _, err := writeFunc(fakeTLSWritePrefix); err != nil {
			return sum, err
		}

		binary.BigEndian.PutUint16(size[:], uint16(chunkSize))

		if _, err := writeFunc(size[:]); err != nil {
			return sum, err
		}

		n, err := writeFunc(p[:chunkSize])
		sum += n

		if err != nil {
			return sum, err
		}

		p = p[chunkSize:]
	}

	return sum, nil
}

func (w *wrapperFakeTLS) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperFakeTLS) Logger() *zap.SugaredLogger {
	return w.parent.Logger().Named("faketls")
}

func (w *wrapperFakeTLS) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperFakeTLS) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func (w *wrapperFakeTLS) Close() error {
	return w.parent.Close()
}

func NewFakeTLS(socket conntypes.StreamReadWriteCloser) conntypes.StreamReadWriteCloser {
	faketls := &wrapperFakeTLS{
		parent: socket,
	}

	faketls.readFunc = func() ([]byte, error) {
		data := &bytes.Buffer{}
		buf := [2]byte{}
		recordType := byte(faketlsRecordTypeCCS)

		for recordType == faketlsRecordTypeCCS {
			if _, err := io.ReadFull(faketls.parent, buf[:1]); err != nil {
				return nil, fmt.Errorf("cannot read record type: %w", err)
			}

			switch buf[0] {
			case faketlsRecordTypeCCS, faketlsRecordTypeApplicationData:
				recordType = buf[0]
			default:
				return nil, fmt.Errorf("incorrect record type %v", buf[0])
			}

			if _, err := io.ReadFull(faketls.parent, buf[:]); err != nil {
				return nil, fmt.Errorf("cannot read version: %w", err)
			}

			if !bytes.Equal(buf[:], []byte{0x03, 0x03}) {
				return nil, fmt.Errorf("unknown tls version %v", buf)
			}

			if _, err := io.ReadFull(faketls.parent, buf[:]); err != nil {
				return nil, fmt.Errorf("cannot read data length: %w", err)
			}

			dataLength := binary.BigEndian.Uint16(buf[:])
			if _, err := io.CopyN(data, faketls.parent, int64(dataLength)); err != nil {
				return nil, fmt.Errorf("cannot copy frame data: %w", err)
			}
		}

		return data.Bytes(), nil
	}

	return faketls
}
