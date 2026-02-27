package network_test

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/network/v2"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DNSTestSuite struct {
	suite.Suite
}

func (suite *DNSTestSuite) TestDefault() {
	resolver, err := network.GetDNS(nil)
	suite.NoError(err)
	suite.doTest(resolver)
}

func (suite *DNSTestSuite) TestDoH() {
	for _, addr := range []string{"1.1.1.1", "cloudflare-dns.com"} {
		suite.Run(addr, func() {
			u, err := url.Parse("https://" + addr)
			require.NoError(suite.T(), err)

			resolver, err := network.GetDNS(u)
			suite.NoError(err)
			suite.doTest(resolver)
		})
	}
}

func (suite *DNSTestSuite) TestDoT() {
	u, err := url.Parse("tls://dns.google")
	require.NoError(suite.T(), err)

	resolver, err := network.GetDNS(u)
	suite.NoError(err)
	suite.doTest(resolver)
}

func (suite *DNSTestSuite) TestUDP() {
	u, err := url.Parse("8.8.8.8")
	require.NoError(suite.T(), err)

	resolver, err := network.GetDNS(u)
	suite.NoError(err)
	suite.doTest(resolver)
}

func (suite *DNSTestSuite) doTest(resolver *net.Resolver) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err := resolver.LookupIP(ctx, "ip4", "dns.google")
	suite.NoError(err)
	suite.Greater(len(ips), 0)
}

func TestGetDNS(t *testing.T) {
	t.Parallel()
	suite.Run(t, &DNSTestSuite{})
}
