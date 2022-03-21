package files_test

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/ipblocklist/files"
	"github.com/stretchr/testify/suite"
)

type MemTestSuite struct {
	suite.Suite
}

func (suite *MemTestSuite) TestOk() {
	_, network1, _ := net.ParseCIDR("192.168.0.1/24")
	_, network2, _ := net.ParseCIDR("2001:0db8:85a3:0000:0000:8a2e:0370:7334/36")

	file := files.NewMem([]*net.IPNet{
		network1,
		network2,
	})

	reader, err := file.Open(context.Background())
	suite.NoError(err)

	data, err := io.ReadAll(reader)
	suite.NoError(err)

	strData := strings.TrimSpace(string(data))

	suite.Contains(strData, "192.168.0.0/24")
	suite.Contains(strData, "2001:db8:8000::/36")
}

func TestMem(t *testing.T) {
	t.Parallel()
	suite.Run(t, &MemTestSuite{})
}
