package cli

import (
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientMSSPlan(t *testing.T) {
	cases := []struct {
		name         string
		network      string
		wantHello    int
		wantBytes    int
		wantListener int
	}{
		{name: "disabled", network: ""},
		{name: "ServerHello at 92, session at 1400", network: "client-mss = 92\n", wantHello: 92, wantListener: 1400},
		{name: "custom bulk", network: "client-mss = 92\nclient-mss-bulk = 1200\n", wantHello: 92, wantListener: 1200},
		{name: "whole session at 92", network: "client-mss = 92\nclient-mss-bulk = 0\n", wantListener: 92},
		{
			name:      "first 500 bytes at 92",
			network:   "client-mss = 92\nclient-mss-bytes = 500\n",
			wantHello: 92, wantBytes: 500, wantListener: 1400,
		},
		{
			name:         "bytes are ignored for the whole session at 92",
			network:      "client-mss = 92\nclient-mss-bulk = 0\nclient-mss-bytes = 500\n",
			wantListener: 92,
		},
		{name: "bytes without client-mss", network: "client-mss-bytes = 500\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conf, err := config.Parse([]byte(`bind-to = "0.0.0.0:443"
secret = "7mqFMMq3P2Tvvt_rPx5qhmFnb29nbGUuY29t"

[network]
` + tc.network))
			require.NoError(t, err)
			require.NoError(t, conf.Validate())

			hello, helloBytes, listener := clientMSSPlan(conf, logger.NewNoopLogger())

			if !network.ListenerMSSSupported {
				assert.Zero(t, hello)
				assert.Zero(t, helloBytes)
				assert.Zero(t, listener)

				return
			}

			assert.Equal(t, tc.wantHello, hello)
			assert.Equal(t, tc.wantBytes, helloBytes)
			assert.Equal(t, tc.wantListener, listener)
		})
	}
}
