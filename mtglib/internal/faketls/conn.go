package faketls

import (
	"bytes"
	"fmt"
	"math/rand"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/internal/faketls/record"
)

type Conn struct {
	essentials.Conn

	readBuffer bytes.Buffer
}

func (c *Conn) Read(p []byte) (int, error) {
	if n, _ := c.readBuffer.Read(p); n > 0 {
		return n, nil
	}

	rec := record.AcquireRecord()
	defer record.ReleaseRecord(rec)

	for {
		if err := rec.Read(c.Conn); err != nil {
			return 0, err //nolint: wrapcheck
		}

		switch rec.Type { //nolint: exhaustive
		case record.TypeApplicationData:
			rec.Payload.WriteTo(&c.readBuffer) //nolint: errcheck

			return c.readBuffer.Read(p) //nolint: wrapcheck
		case record.TypeChangeCipherSpec:
		default:
			return 0, fmt.Errorf("unsupported record type %v", rec.Type)
		}
	}
}

func (c *Conn) Write(p []byte) (int, error) {
	rec := record.AcquireRecord()
	defer record.ReleaseRecord(rec)

	rec.Type = record.TypeApplicationData
	rec.Version = record.Version12

	sendBuffer := acquireBytesBuffer()
	defer releaseBytesBuffer(sendBuffer)

	lenP := len(p)

	for len(p) > 0 {
		chunkSize := rand.Intn(record.TLSMaxRecordSize)
		if chunkSize > len(p) || chunkSize == 0 {
			chunkSize = len(p)
		}

		rec.Payload.Reset()
		rec.Payload.Write(p[:chunkSize])
		rec.Dump(sendBuffer) //nolint: errcheck

		p = p[chunkSize:]
	}

	if _, err := c.Conn.Write(sendBuffer.Bytes()); err != nil {
		return 0, err //nolint: wrapcheck
	}

	return lenP, nil
}
