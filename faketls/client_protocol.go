package faketls

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/9seconds/mtg/antireplay"
	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/protocol"
	"github.com/9seconds/mtg/stats"
	"github.com/9seconds/mtg/tlstypes"
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
			rewinded.Rewind()
			c.cloakHost(rewinded)

			return nil, errors.New("failed first bytes of tls handshake")
		}
	}

	rewinded.Rewind()
	rewinded = stream.NewRewind(rewinded)

	if err := c.tlsHandshake(rewinded); err != nil {
		rewinded.Rewind()
		c.cloakHost(rewinded)

		return nil, fmt.Errorf("failed tls handshake: %w", err)
	}

	conn := stream.NewFakeTLS(socket)
	conn, err := c.ClientProtocol.Handshake(conn)

	if err != nil {
		return nil, err
	}

	return conn, err
}

func (c *ClientProtocol) tlsHandshake(conn io.ReadWriter) error {
	helloRecord, err := tlstypes.ReadRecord(conn)
	if err != nil {
		return fmt.Errorf("cannot read initial record: %w", err)
	}

	buf := acquireBytesBuffer()
	defer releaseBytesBuffer(buf)

	helloRecord.Data.WriteBytes(buf)

	clientHello, err := tlstypes.ParseClientHello(buf.Bytes())
	if err != nil {
		return fmt.Errorf("cannot parse client hello: %w", err)
	}

	digest := clientHello.Digest()
	for i := 0; i < len(digest)-4; i++ {
		if digest[i] != 0 {
			return errBadDigest
		}
	}

	timestamp := int64(binary.LittleEndian.Uint32(digest[len(digest)-4:]))
	createdAt := time.Unix(timestamp, 0)
	timeDiff := time.Since(createdAt)

	if (timeDiff > TimeSkew || timeDiff < -TimeSkew) && timestamp > TimeFromBoot {
		return errBadTime
	}

	if antireplay.Cache.HasTLS(clientHello.Random[:]) {
		stats.Stats.ReplayDetected()
		return errors.New("replay attack is detected")
	}

	antireplay.Cache.AddTLS(clientHello.Random[:])
	serverHello := tlstypes.NewServerHello(clientHello)
	serverHelloPacket := serverHello.WelcomePacket()

	if _, err := conn.Write(serverHelloPacket); err != nil {
		return fmt.Errorf("cannot send welcome packet: %w", err)
	}

	return nil
}

func (c *ClientProtocol) cloakHost(clientConn io.ReadWriteCloser) {
	stats.Stats.CloakedRequest()

	addr := net.JoinHostPort(config.C.CloakHost, strconv.Itoa(config.C.CloakPort))
	hostConn, err := net.Dial("tcp", addr)

	if err != nil {
		return
	}

	cloak(clientConn, hostConn)
}

func MakeClientProtocol() protocol.ClientProtocol {
	return &ClientProtocol{}
}
