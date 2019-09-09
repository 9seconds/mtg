package rpc

import "github.com/9seconds/mtg/mtproto"

var HandshakeRequest = append(mtproto.TagHandshake,
	append(mtproto.HandshakeFlags,
		append(mtproto.HandshakeSenderPID, mtproto.HandshakePeerPID...)...)...)
