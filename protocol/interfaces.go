package protocol

import "mtg/conntypes"

type ClientProtocol interface {
	Handshake(conntypes.StreamReadWriteCloser) (conntypes.StreamReadWriteCloser, error)
	ConnectionType() conntypes.ConnectionType
	ConnectionProtocol() conntypes.ConnectionProtocol
	DC() conntypes.DC
}

type ClientProtocolMaker func() ClientProtocol
