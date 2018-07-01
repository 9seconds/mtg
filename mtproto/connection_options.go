package mtproto

import (
	"bytes"

	"github.com/juju/errors"
)

// ConnectionType is a type of obfuscated2/mtproto connection requested
// by the user.
type ConnectionType uint8

type ConnectionProtocol uint8

// ConnectionOpts presents an options, metadata on connection requested
// by the user on handshake.
type ConnectionOpts struct {
	DC              int16
	ConnectionType  ConnectionType
	ConnectionProto ConnectionProtocol
}

// Different connection types which user requests from Telegram.
const (
	ConnectionTypeUnknown ConnectionType = iota
	ConnectionTypeAbridged
	ConnectionTypeIntermediate
)

const (
	ConnectionProtocolIPv4 ConnectionProtocol = 1
	ConnectionProtocolIPv6                    = ConnectionProtocolIPv4 << 1
	ConnectionProtocolAny                     = ConnectionProtocolIPv4 | ConnectionProtocolIPv6
)

// Connection tags for mtproto handshakes.
var (
	ConnectionTagAbridged     = []byte{0xef, 0xef, 0xef, 0xef}
	ConnectionTagIntermediate = []byte{0xee, 0xee, 0xee, 0xee}
)

// Tag maps connection type to the corresponding handshake tag.
func (t ConnectionType) Tag() ([]byte, error) {
	switch t {
	case ConnectionTypeAbridged:
		return ConnectionTagAbridged, nil
	case ConnectionTypeIntermediate:
		return ConnectionTagIntermediate, nil
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

	return ConnectionTypeUnknown, errors.New("Unknown handshake protocol")
}
