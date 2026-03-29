//go:build !mips && !mipsle

package relay

import "github.com/9seconds/mtg/v2/mtglib/internal/tls"

const (
	bufPoolSize = tls.MaxRecordPayloadSize
)
