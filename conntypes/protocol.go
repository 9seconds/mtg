package conntypes

type ConnectionProtocol uint8

func (c ConnectionProtocol) String() string {
	switch c {
	case ConnectionProtocolAny:
		return "any"
	case ConnectionProtocolIPv4:
		return "ipv4"
	}

	return "ipv6"
}

const (
	ConnectionProtocolIPv4 ConnectionProtocol = 1
	ConnectionProtocolIPv6                    = ConnectionProtocolIPv4 << 1
	ConnectionProtocolAny                     = ConnectionProtocolIPv4 | ConnectionProtocolIPv6
)
