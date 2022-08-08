// Package ipblocklist contains default implementation of the
// [mtglib.IPBlocklist] for mtg.
//
// Please check documentation for [mtglib.IPBlocklist] interface to get an idea
// of this abstraction.
package ipblocklist

import "time"

const (
	// DefaultFireholDownloadConcurrency defines a default max number of
	// concurrent downloads of ip blocklists for Firehol.
	DefaultFireholDownloadConcurrency = 1

	// DefaultFireholUpdateEach defines a default time period when Firehol
	// requests updates of the blocklists.
	DefaultFireholUpdateEach = 6 * time.Hour
)
