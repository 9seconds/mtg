//go:build mips || mipsle

package relay

import "github.com/dolonet/mtg-multi/mtglib/internal/tls"

const (
	// MIPS is quite short in resources, and usually it means that it will run
	// on Microtiks, OpenWRT-based routers or similar hardware. I think it worth
	// to sacrifice a number of read syscalls (read, CPU load) to shrink
	// limited RAM resources.
	bufPoolSize = tls.MaxRecordPayloadSize / 2
)
