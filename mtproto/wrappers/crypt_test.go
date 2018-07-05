package wrappers

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/9seconds/mtg/mtproto/rpc"
)

var proxySecret = []byte{196, 249, 250, 202, 150, 120, 230, 187, 72, 173,
	108, 126, 44, 229, 192, 210, 68, 48, 100, 93, 85, 74, 221, 235, 85, 65,
	158, 3, 77, 166, 39, 33, 208, 70, 234, 171, 110, 82, 171, 20, 169, 90, 68,
	62, 207, 179, 70, 62, 121, 160, 90, 102, 97, 42, 223, 156, 174, 218, 139,
	233, 168, 13, 166, 152, 111, 176, 166, 255, 56, 122, 248, 77, 136, 239,
	58, 100, 19, 113, 62, 92, 51, 119, 246, 225, 163, 212, 125, 153, 245, 224,
	197, 110, 236, 232, 240, 92, 84, 196, 144, 176, 121, 227, 27, 239, 130,
	255, 14, 232, 242, 176, 163, 39, 86, 210, 73, 197, 242, 18, 105, 129, 108,
	183, 6, 27, 38, 93, 178, 18}

func TestMakeKeys(t *testing.T) {
	req, err := rpc.NewNonceRequest(proxySecret)
	assert.Nil(t, err)

	copy(req.Nonce[:], []byte{24, 49, 53, 111, 198, 10, 235, 180, 230, 112, 92, 78, 1, 201, 106, 105})
	binary.LittleEndian.PutUint32(req.CryptoTS[:], 1528396015)

	resp := &rpc.NonceResponse{}
	copy(resp.Nonce[:], []byte{247, 40, 210, 56, 65, 12, 101, 170, 216, 155, 14, 253, 250, 238, 219, 226})

	cltAddr := &net.TCPAddr{
		IP:   net.ParseIP("80.211.29.34"),
		Port: 54208,
	}
	srvAddr := &net.TCPAddr{
		IP:   net.ParseIP("149.154.162.38"),
		Port: 80,
	}

	key, iv := makeKeys(CipherPurposeClient, req, resp, cltAddr, srvAddr, proxySecret)
	assert.Equal(t, key, []byte{165, 158, 127, 49, 41, 232, 187, 69, 38, 29, 163, 226, 183, 146, 28, 67, 225, 224, 134, 191, 207, 152, 255, 166, 152, 66, 169, 196, 54, 135, 50, 188})
	assert.Equal(t, iv, []byte{33, 110, 125, 221, 183, 121, 160, 116, 130, 180, 156, 249, 52, 111, 37, 178})
}
