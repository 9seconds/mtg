package conntypes

type ConnectionType uint8

const (
	ConnectionTypeUnknown ConnectionType = iota
	ConnectionTypeAbridged
	ConnectionTypeIntermediate
	ConnectionTypeSecure
)

var (
	ConnectionTagAbridged     = []byte{0xef, 0xef, 0xef, 0xef}
	ConnectionTagIntermediate = []byte{0xee, 0xee, 0xee, 0xee}
	ConnectionTagSecure       = []byte{0xdd, 0xdd, 0xdd, 0xdd}
)

func (t ConnectionType) Tag() []byte {
	switch t {
	case ConnectionTypeAbridged:
		return ConnectionTagAbridged
	case ConnectionTypeIntermediate:
		return ConnectionTagIntermediate
	case ConnectionTypeSecure, ConnectionTypeUnknown:
		return ConnectionTagSecure
	}

	return ConnectionTagSecure
}
