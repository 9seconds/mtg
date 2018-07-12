package rpc

import (
	"bytes"

	"github.com/juju/errors"
)

type ProxyResponseType uint8

const (
	ProxyResponseTypeAns ProxyResponseType = iota
	ProxyResponseTypeSimpleAck
	ProxyResponseTypeCloseExt
)

type ProxyResponse interface {
	ConnectionID() []byte
	Data() []byte
	ResponseType() ProxyResponseType
}

type BaseProxyResponse struct {
	respType     ProxyResponseType
	connectionID []byte
	data         []byte
}

func (b *BaseProxyResponse) ConnectionID() []byte {
	return b.connectionID
}

func (b *BaseProxyResponse) Data() []byte {
	return b.data
}

func (b *BaseProxyResponse) ResponseType() ProxyResponseType {
	return b.respType
}

type ProxyAns struct {
	BaseProxyResponse

	Flags []byte
}

type ProxySimpleAck struct {
	BaseProxyResponse
}

type ProxyCloseExt struct {
	BaseProxyResponse
}

func GetProxyResponse(data []byte) (ProxyResponse, error) {
	if len(data) < 4 {
		return nil, errors.Errorf("Response length is less than minimal length: %d", len(data))
	}

	tag, data := data[:4], data[4:]
	switch {
	case bytes.Equal(tag, TagProxyAns):
		return makeProxyAns(data)
	case bytes.Equal(tag, TagSimpleAck):
		return makeSimpleAck(data)
	case bytes.Equal(tag, TagCloseExt):
		return makeCloseExt(data)
	}

	return nil, errors.Errorf("Unknown RPC answer %v", tag)
}

func makeProxyAns(data []byte) (ProxyResponse, error) {
	if len(data) < 12 {
		return nil, errors.Errorf("Incorrect length of proxy answer: %d", len(data))
	}

	return &ProxyAns{
		BaseProxyResponse: BaseProxyResponse{
			connectionID: data[4:12],
			data:         data[12:],
			respType:     ProxyResponseTypeAns,
		},
		Flags: data[:4],
	}, nil
}

func makeSimpleAck(data []byte) (ProxyResponse, error) {
	if len(data) != 12 {
		return nil, errors.Errorf("Incorrect length of simple ack: %d", len(data))
	}

	return &ProxySimpleAck{
		BaseProxyResponse: BaseProxyResponse{
			connectionID: data[:8],
			data:         data[8:],
			respType:     ProxyResponseTypeSimpleAck,
		},
	}, nil
}

func makeCloseExt(data []byte) (ProxyResponse, error) {
	if len(data) != 8 {
		return nil, errors.Errorf("Incorrect length of close external: %d", len(data))
	}

	return &ProxyCloseExt{
		BaseProxyResponse: BaseProxyResponse{
			connectionID: data,
			respType:     ProxyResponseTypeCloseExt,
		},
	}, nil
}
