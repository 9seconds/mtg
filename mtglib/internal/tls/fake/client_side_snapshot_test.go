package fake_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dolonet/mtg-multi/mtglib"
	"github.com/dolonet/mtg-multi/mtglib/internal/tls/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type clientHelloSnapshot struct {
	Time        int    `json:"time"`
	Random      string `json:"random"`
	SessionID   string `json:"sessionId"`
	Host        string `json:"host"`
	CipherSuite int    `json:"cipherSuite"`
	Full        string `json:"full"`
}

func (c clientHelloSnapshot) GetRandom() []byte {
	data, _ := base64.StdEncoding.DecodeString(c.Random)

	return data
}

func (c clientHelloSnapshot) GetSessionID() []byte {
	data, _ := base64.StdEncoding.DecodeString(c.SessionID)

	return data
}

func (c clientHelloSnapshot) GetCipherSuite() uint16 {
	return uint16(c.CipherSuite)
}

func (c clientHelloSnapshot) GetFull() []byte {
	data, _ := base64.StdEncoding.DecodeString(c.Full)

	return data
}

type ParseClientHelloSnapshotTestSuite struct {
	suite.Suite

	secret mtglib.Secret
}

func (suite *ParseClientHelloSnapshotTestSuite) SetupSuite() {
	parsed, err := mtglib.ParseSecret(
		"ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d",
	)
	require.NoError(suite.T(), err)

	suite.secret = parsed
}

func (suite *ParseClientHelloSnapshotTestSuite) makeConn(data []byte) *parseClientHelloConnMock {
	readBuf := &bytes.Buffer{}
	readBuf.Write(data)

	connMock := &parseClientHelloConnMock{
		readBuf: readBuf,
	}

	return connMock
}

func (suite *ParseClientHelloSnapshotTestSuite) TestSnapshotOk() {
	files, err := os.ReadDir("testdata")
	require.NoError(suite.T(), err)

	for _, v := range files {
		if !strings.HasPrefix(v.Name(), "client-hello-ok") {
			continue
		}

		path := filepath.Join("testdata", v.Name())

		suite.T().Run(v.Name(), func(t *testing.T) {
			fileData, err := os.ReadFile(path)
			assert.NoError(t, err)

			snapshot := &clientHelloSnapshot{}
			assert.NoError(t, json.Unmarshal(fileData, snapshot))

			connMock := suite.makeConn(snapshot.GetFull())
			defer connMock.AssertExpectations(t)

			hello, err := fake.ReadClientHello(
				connMock,
				suite.secret.Key[:],
				suite.secret.Host,
				TolerateTime,
			)
			require.NoError(t, err)

			assert.Equal(t, snapshot.GetRandom(), hello.Random[:])
			assert.Equal(t, snapshot.GetSessionID(), hello.SessionID)
			assert.Equal(t, snapshot.GetCipherSuite(), hello.CipherSuite)
		})
	}
}

func (suite *ParseClientHelloSnapshotTestSuite) TestSnapshotBad() {
	files, err := os.ReadDir("testdata")
	require.NoError(suite.T(), err)

	for _, v := range files {
		if !strings.HasPrefix(v.Name(), "client-hello-bad") {
			continue
		}

		path := filepath.Join("testdata", v.Name())

		suite.T().Run(v.Name(), func(t *testing.T) {
			fileData, err := os.ReadFile(path)
			assert.NoError(t, err)

			snapshot := &clientHelloSnapshot{}
			assert.NoError(t, json.Unmarshal(fileData, snapshot))

			connMock := suite.makeConn(snapshot.GetFull())
			defer connMock.AssertExpectations(t)

			_, err = fake.ReadClientHello(
				connMock,
				suite.secret.Key[:],
				suite.secret.Host,
				TolerateTime,
			)
			assert.ErrorIs(t, err, fake.ErrBadDigest)
		})
	}
}

func TestParseClientHelloSnapshot(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ParseClientHelloSnapshotTestSuite{})
}
