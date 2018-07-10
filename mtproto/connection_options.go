package mtproto

import (
	"bytes"
	"net"

	"github.com/juju/errors"
)

// ConnectionType is a type of obfuscated2/mtproto connection requested
// by the user.
type ConnectionType uint8

// ConnectionProtocol is a type of IP protocol to use.
type ConnectionProtocol uint8

// Hacks is a simple structure to store flags for packet transmission.
type Hacks struct {
	SimpleAck bool
	QuickAck  bool
}

// ConnectionOpts presents an options, metadata on connection requested
// by the user on handshake.
type ConnectionOpts struct {
	DC              int16
	ConnectionType  ConnectionType
	ConnectionProto ConnectionProtocol
	// Read and Write means direction related to the client.
	// ReadHacks are meant to be flushed on client read
	// WriteHacks are meant to be flushed on client write.
	ReadHacks  Hacks
	WriteHacks Hacks
	ClientAddr *net.TCPAddr
}

// Different connection types which user requests from Telegram.
const (
	ConnectionTypeUnknown ConnectionType = iota
	ConnectionTypeAbridged
	ConnectionTypeIntermediate
	ConnectionTypeSecure
)

// ConnectionProtocol* define which connection protocols to use.
// ConnectionProtocolAny means that any is suitable.
const (
	ConnectionProtocolIPv4 ConnectionProtocol = 1
	ConnectionProtocolIPv6                    = ConnectionProtocolIPv4 << 1
	ConnectionProtocolAny                     = ConnectionProtocolIPv4 | ConnectionProtocolIPv6
)

// Connection tags for mtproto handshakes.
var (
	ConnectionTagAbridged     = []byte{0xef, 0xef, 0xef, 0xef}
	ConnectionTagIntermediate = []byte{0xee, 0xee, 0xee, 0xee}
	ConnectionTagSecure       = []byte{0xdd, 0xdd, 0xdd, 0xdd}
)

// Tag maps connection type to the corresponding handshake tag.
func (t ConnectionType) Tag() ([]byte, error) {
	switch t {
	case ConnectionTypeAbridged:
		return ConnectionTagAbridged, nil
	case ConnectionTypeIntermediate:
		return ConnectionTagIntermediate, nil
	case ConnectionTypeSecure:
		return ConnectionTagSecure, nil
	default:
		return nil, errors.Errorf("Unknown connection type %d", t)
	}
}

// ConnectionTagFromHandshake maps magic bytes to the connection type.
func ConnectionTagFromHandshake(magic []byte) (ConnectionType, error) {
	if bytes.Equal(magic, ConnectionTagIntermediate) {
		return ConnectionTypeIntermediate, nil
	}
	if bytes.Equal(magic, ConnectionTagAbridged) {
		return ConnectionTypeAbridged, nil
	}
	if bytes.Equal(magic, ConnectionTagSecure) {
		return ConnectionTypeSecure, nil
	}

	return ConnectionTypeUnknown, errors.New("Unknown handshake protocol")
}
