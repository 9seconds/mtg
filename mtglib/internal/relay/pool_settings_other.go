//go:build !mips && !mipsle

package relay

import "github.com/dolonet/mtg-multi/mtglib/internal/tls"

const (
	bufPoolSize = tls.MaxRecordPayloadSize
)
