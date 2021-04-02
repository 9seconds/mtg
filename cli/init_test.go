package cli_test

import (
	"net/http"

	"github.com/9seconds/mtg/v2/cli"
	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type CommonTestSuite struct {
	suite.Suite

	cli         *cli.CLI
	networkMock *testlib.MtglibNetworkMock
	httpClient  *http.Client
}

func (suite *CommonTestSuite) SetupTest() {
	suite.networkMock = &testlib.MtglibNetworkMock{}
	suite.httpClient = &http.Client{}
	suite.cli = &cli.CLI{}

	httpmock.ActivateNonDefault(suite.httpClient)

	suite.networkMock.
		On("MakeHTTPClient", mock.Anything).
		Maybe().
		Return(suite.httpClient)
}

func (suite *CommonTestSuite) TearDownTest() {
	suite.networkMock.AssertExpectations(suite.T())
	httpmock.DeactivateAndReset()
}
