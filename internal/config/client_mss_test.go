package config_test

import (
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseClientMSS(t *testing.T, network string) (*config.Config, error) {
	t.Helper()

	conf, err := config.Parse([]byte(`bind-to = "0.0.0.0:443"
secret = "7mqFMMq3P2Tvvt_rPx5qhmFnb29nbGUuY29t"

[network]
` + network))
	require.NoError(t, err)

	return conf, conf.Validate()
}

func TestClientMSS(t *testing.T) {
	cases := []struct {
		name          string
		network       string
		wantHandshake uint
		wantBulk      uint
	}{
		{name: "not set", network: "", wantHandshake: 0, wantBulk: 0},
		{name: "explicit zero", network: "client-mss = 0", wantHandshake: 0, wantBulk: 0},
		{name: "default bulk 1400", network: "client-mss = 92", wantHandshake: 92, wantBulk: 1400},
		{name: "custom bulk", network: "client-mss = 92\nclient-mss-bulk = 1200", wantHandshake: 92, wantBulk: 1200},
		{name: "bulk 0 keeps the whole session at client-mss", network: "client-mss = 92\nclient-mss-bulk = 0", wantHandshake: 92, wantBulk: 0},
		{name: "bulk without client-mss has no effect", network: "client-mss-bulk = 1400", wantHandshake: 0, wantBulk: 0},
		{name: "lower bound", network: "client-mss = 48", wantHandshake: 48, wantBulk: 1400},
		{name: "upper bounds", network: "client-mss = 1460\nclient-mss-bulk = 65495", wantHandshake: 1460, wantBulk: 65495},
		{name: "bulk lower bound", network: "client-mss = 92\nclient-mss-bulk = 536", wantHandshake: 92, wantBulk: 536},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conf, err := parseClientMSS(t, tc.network)
			require.NoError(t, err)

			handshake, bulk := conf.GetClientMSS()
			assert.Equal(t, tc.wantHandshake, handshake)
			assert.Equal(t, tc.wantBulk, bulk)
		})
	}
}

func TestClientMSSValidation(t *testing.T) {
	cases := []struct {
		name    string
		network string
		wantErr string
	}{
		{name: "client-mss below 48", network: "client-mss = 47", wantErr: "network.client-mss must be 0 or 48..1460"},
		{name: "client-mss above 1460", network: "client-mss = 1461", wantErr: "network.client-mss must be 0 or 48..1460"},
		{name: "bulk below 536", network: "client-mss = 92\nclient-mss-bulk = 535", wantErr: "network.client-mss-bulk must be 0 or 536..65495"},
		{name: "bulk above 65495", network: "client-mss = 92\nclient-mss-bulk = 65496", wantErr: "network.client-mss-bulk must be 0 or 536..65495"},
		{name: "bulk not greater than client-mss", network: "client-mss = 1400\nclient-mss-bulk = 1400", wantErr: "must be greater than network.client-mss"},
		{name: "bulk 0 and client-mss below the kernel minimum", network: "client-mss = 60\nclient-mss-bulk = 0", wantErr: "at least 88"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseClientMSS(t, tc.network)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestClientMSSNotANumber(t *testing.T) {
	_, err := config.Parse([]byte(`bind-to = "0.0.0.0:443"
secret = "7mqFMMq3P2Tvvt_rPx5qhmFnb29nbGUuY29t"

[network]
client-mss = "small"
`))
	assert.Error(t, err)
}
