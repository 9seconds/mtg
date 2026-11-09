package utils_test

import (
	"net"
	"testing"

	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type notTCPListener struct {
	net.Listener
}

func TestSetListenerMSSZeroIsNoop(t *testing.T) {
	assert.NoError(t, utils.SetListenerMSS(notTCPListener{}, 0))
}

func TestSetListenerMSSRejectsNonTCPListener(t *testing.T) {
	assert.Error(t, utils.SetListenerMSS(notTCPListener{}, 92))
}

func TestSetListenerMSSAcceptsPlainTCPListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer listener.Close() //nolint: errcheck

	assert.NoError(t, utils.SetListenerMSS(listener, 1400))
}
