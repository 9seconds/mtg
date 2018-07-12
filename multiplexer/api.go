package multiplexer

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/mtproto/rpc"
)

const readTimeout = 20 * time.Second

func Register(connID ConnectionID) (<-chan rpc.ProxyResponse, error) {
	return instance.register(connID)
}

func Write(data []byte, opts *mtproto.ConnectionOpts, connID ConnectionID) {
	instance.writeChan <- &Request{
		data:     data,
		protocol: opts.ConnectionProto,
		dc:       opts.DC,
		connID:   connID,
	}
}

func Read(channel <-chan rpc.ProxyResponse) (rpc.ProxyResponse, error) {
	select {
	case resp, ok := <-channel:
		if !ok {
			return nil, io.EOF
		}
		return resp, nil
	case <-time.After(readTimeout):
		return nil, errors.New("Cannot read from channel in time")
	}
}

func Deregister(connID ConnectionID) {
	instance.deregister(connID)
}

func ToConnectionID(data []byte) (ConnectionID, error) {
	if len(data) != 8 {
		return 0, errors.Errorf("Incorrect data length %d", len(data))
	}

	return ConnectionID(binary.LittleEndian.Uint64(data)), nil
}
