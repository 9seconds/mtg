package tlstypes

import (
	"container/ring"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/config"
)

const (
	connectionServerKeepCertificates = 5
	connectionServerUpdateEvery      = 10 * time.Minute
)

type connectionServer struct {
	nextWriteItem *ring.Ring
	nextReadItem  *ring.Ring

	ctx        context.Context
	channelGet chan chan<- *x509.Certificate
}

func (c *connectionServer) fetch() (*x509.Certificate, error) {
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

	return certificates[0], nil
}

func (c *connectionServer) run() {
	logger := zap.S().Named("tls-connection-server")

	ticker := time.NewTicker(connectionServerUpdateEvery)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case resp := <-c.channelGet:
			resp <- c.nextReadItem.Value.(*x509.Certificate)
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
