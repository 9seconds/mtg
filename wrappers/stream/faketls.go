package stream

import (
	"errors"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"

	"mtg/conntypes"
	"mtg/tlstypes"
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
		return 0, errors.New("timeout")
	})
}

func (w *wrapperFakeTLS) write(p []byte, writeFunc func([]byte) (int, error)) (int, error) {
	sum := 0

	for _, v := range tlstypes.MakeRecords(p) {
		_, err := writeFunc(v.Bytes())
		if err != nil {
			return sum, err
		}

		sum += len(v.Data.Bytes())
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
		for {
			rec, err := tlstypes.ReadRecord(faketls.parent)
			if err != nil {
				return nil, err
			}

			switch rec.Type {
			case tlstypes.RecordTypeChangeCipherSpec:
			case tlstypes.RecordTypeApplicationData:
				return rec.Data.Bytes(), nil
			default:
				return nil, fmt.Errorf("unsupported record type %v", rec.Type)
			}
		}
	}

	return faketls
}
