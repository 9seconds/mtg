package dc_test

import (
	"testing"

	"github.com/9seconds/mtg/v2/mtglib/internal/dc"
	"github.com/stretchr/testify/assert"
)

func TestAddr(t *testing.T) {
	t.Parallel()

	addr := dc.Addr{Network: "tcp4", Address: "127.0.0.1:443"}

	assert.Equal(t, "127.0.0.1:443", addr.String())
}
