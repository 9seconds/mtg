package network

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type DNSResolverTestSuite struct {
	suite.Suite

	d *dnsResolver
}

func (suite *DNSResolverTestSuite) TestLookupA() {
	suite.d.LookupA("google.com")
	time.Sleep(10 * time.Millisecond)

	addrs := suite.d.LookupA("google.com")

	for _, v := range addrs {
		suite.NotEmpty(v)
		suite.NotNil(net.ParseIP(v).To4())
	}
}

func (suite *DNSResolverTestSuite) TestLookupAAAA() {
	suite.d.LookupAAAA("google.com")
	time.Sleep(10 * time.Millisecond)

	addrs := suite.d.LookupAAAA("google.com")

	for _, v := range addrs {
		suite.NotEmpty(v)
		suite.Nil(net.ParseIP(v).To4())
		suite.NotNil(net.ParseIP(v).To16())
	}
}

func (suite *DNSResolverTestSuite) SetupTest() {
	suite.d = newDNSResolver("1.1.1.1", &http.Client{})
}

func TestDNSResolver(t *testing.T) {
	t.Parallel()
	suite.Run(t, &DNSResolverTestSuite{})
}
