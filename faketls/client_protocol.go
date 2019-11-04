package faketls

import (
	"bufio"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/wrappers/stream"
)

type ClientProtocol struct {
	obfuscated2.ClientProtocol
}

func (c *ClientProtocol) Handshake(socket conntypes.StreamReadWriteCloser) (conntypes.StreamReadWriteCloser, error) {
	rewinded := stream.NewRewind(socket)
	bufferedReader := bufio.NewReader(rewinded)

	for _, expected := range faketlsStartBytes {
		if actual, err := bufferedReader.ReadByte(); err != nil || actual != expected {
			return nil, c.simulateWebsite(rewinded)
		}
	}

	if err := c.tlsHandshake(rewinded); err != nil {
		return nil, c.simulateWebsite(rewinded)
	}

	conn, err := c.ClientProtocol.Handshake(socket)
	if err != nil {
		return nil, err
	}

	return conn, err
}
