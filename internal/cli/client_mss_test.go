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
		wantListener int
	}{
		{name: "disabled", network: "", wantHello: 0, wantListener: 0},
		{name: "ServerHello at 92, session at 1400", network: "client-mss = 92\n", wantHello: 92, wantListener: 1400},
		{name: "custom bulk", network: "client-mss = 92\nclient-mss-bulk = 1200\n", wantHello: 92, wantListener: 1200},
		{name: "whole session at 92", network: "client-mss = 92\nclient-mss-bulk = 0\n", wantHello: 0, wantListener: 92},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conf, err := config.Parse([]byte(`bind-to = "0.0.0.0:443"
secret = "7mqFMMq3P2Tvvt_rPx5qhmFnb29nbGUuY29t"

[network]
` + tc.network))
			require.NoError(t, err)
			require.NoError(t, conf.Validate())

			hello, listener := clientMSSPlan(conf, logger.NewNoopLogger())

			if !network.ListenerMSSSupported {
				assert.Zero(t, hello)
				assert.Zero(t, listener)

				return
			}

			assert.Equal(t, tc.wantHello, hello)
			assert.Equal(t, tc.wantListener, listener)
		})
	}
}
