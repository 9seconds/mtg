package rpc

import (
	"bytes"
	"fmt"

	"mtg/conntypes"
)

type ProxyResponseType uint8

const (
	ProxyResponseTypeAns ProxyResponseType = iota
	ProxyResponseTypeSimpleAck
	ProxyResponseTypeCloseExt
)

type ProxyResponse struct {
	Type    ProxyResponseType
	ConnID  conntypes.ConnID
	Payload conntypes.Packet
}

func ParseProxyResponse(packet conntypes.Packet) (*ProxyResponse, error) {
	var response ProxyResponse

	if len(packet) < 4 {
		return nil, fmt.Errorf("incorrect packet length: %d", len(packet))
	}

	tag := packet[:4]

	switch {
	case bytes.Equal(tag, TagProxyAns):
		response.Type = ProxyResponseTypeAns
		copy(response.ConnID[:], packet[8:16])
		response.Payload = packet[16:]

		return &response, nil
	case bytes.Equal(tag, TagSimpleAck):
		response.Type = ProxyResponseTypeSimpleAck
		copy(response.ConnID[:], packet[4:12])
		response.Payload = packet[12:]

		return &response, nil
	case bytes.Equal(tag, TagCloseExt):
		response.Type = ProxyResponseTypeCloseExt

		return &response, nil
	}

	return nil, fmt.Errorf("unknown response type %x", tag)
}
