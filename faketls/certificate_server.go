package faketls

import (
	"bytes"
	"container/ring"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/config"
)

type connectionServer struct {
	nextWriteItem *ring.Ring
	nextReadItem  *ring.Ring

	ctx        context.Context
	channelGet chan chan<- []byte
}

func (c *connectionServer) get() ([]byte, error) {
	resp := make(chan []byte)
	select {
	case <-c.ctx.Done():
		return nil, errors.New("context closed")
	case c.channelGet <- resp:
		return <-resp, nil
	}
}

func (c *connectionServer) fetch() ([]byte, error) {
	addr := net.JoinHostPort(config.C.CloakHost, strconv.Itoa(config.C.CloakPort))
	conn, err := tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true}) // nolint: gosec

	if err != nil {
		return nil, fmt.Errorf("cannot connect to the masked host: %w", err)
	}

	defer conn.Close()

	if err = conn.Handshake(); err != nil {
		return nil, fmt.Errorf("cannot perform tls handshake: %w", err)
	}

	certificates := conn.ConnectionState().PeerCertificates
	if len(certificates) == 0 {
		return nil, errors.New("no certificates is found")
	}

	var buf bytes.Buffer

	for _, v := range certificates {
		buf.Write(v.Raw)
	}

	return buf.Bytes(), nil
}

func (c *connectionServer) run(tickEvery time.Duration) {
	logger := zap.S().Named("tls-connection-server")

	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case resp := <-c.channelGet:
			resp <- c.nextReadItem.Value.([]byte)
			close(resp)

			c.nextReadItem = c.nextReadItem.Next()
		case <-ticker.C:
			cert, err := c.fetch()
			switch err {
			case nil:
				c.nextWriteItem.Value = cert
				c.nextWriteItem = c.nextWriteItem.Next()
			default:
				logger.Warnw("cannot fetch certificates", "error", err)
			}
		}
	}
}
