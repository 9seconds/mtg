package rpc

var HandshakeRequest = append(TagHandshake,
	append(HandshakeFlags,
		append(HandshakeSenderPID, HandshakePeerPID...)...)...)
