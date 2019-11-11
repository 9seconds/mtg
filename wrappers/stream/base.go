package stream

import (
	"net"

	"mtg/conntypes"
)

func NewClientConn(parent net.Conn, connID conntypes.ConnID) conntypes.StreamReadWriteCloser {
	conn := newConn(parent, connID, connPurposeClient)
	conn = NewTrafficStats(conn)

	return conn
}

func NewTelegramConn(dc conntypes.DC, parent net.Conn) conntypes.StreamReadWriteCloser {
	conn := newConn(parent, conntypes.ConnID{}, connPurposeTelegram)
	conn = NewTelegramStats(dc, conn)

	return conn
}
