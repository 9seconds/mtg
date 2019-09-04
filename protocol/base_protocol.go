package protocol

import "github.com/9seconds/mtg/conntypes"

type BaseProtocol struct {
	ConnectionType     conntypes.ConnectionType
	ConnectionProtocol conntypes.ConnectionProtocol
	DC                 conntypes.DC
}

func (b *BaseProtocol) GetConnectionType() conntypes.ConnectionType {
	return b.ConnectionType
}

func (b *BaseProtocol) GetConnectionProtocol() conntypes.ConnectionProtocol {
	return b.ConnectionProtocol
}

func (b *BaseProtocol) GetDC() conntypes.DC {
	return b.DC
}
