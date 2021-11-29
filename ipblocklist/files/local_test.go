package files_test

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/ipblocklist/files"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type LocalTestSuite struct {
	suite.Suite
}

func (suite *LocalTestSuite) getLocalFile(name string) string {
	return filepath.Join("testdata", name)
}

func (suite *LocalTestSuite) TestIncorrect() {
	names := []string{
		"absent",
		"directory",
	}

	for _, v := range names {
		value := v

		suite.T().Run(v, func(t *testing.T) {
			_, err := files.NewLocal(suite.getLocalFile(value))
			assert.Error(t, err)
		})
	}
}

func (suite *LocalTestSuite) TestOk() {
	file, err := files.NewLocal(suite.getLocalFile("readable"))
	suite.NoError(err)

	reader, err := file.Open(context.Background())
	suite.NoError(err)

	data, err := io.ReadAll(reader)
	suite.NoError(err)

	suite.Equal("Hooray!", strings.TrimSpace(string(data)))
}

func TestLocal(t *testing.T) {
	t.Parallel()
	suite.Run(t, &LocalTestSuite{})
}
