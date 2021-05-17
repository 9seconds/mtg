package faketls_test

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/mtglib/internal/faketls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ClientHelloSnapshot struct {
	Time        int    `json:"time"`
	Random      string `json:"random"`
	SessionID   string `json:"sessionId"`
	Host        string `json:"host"`
	CipherSuite int    `json:"cipherSuite"`
	Full        string `json:"full"`
}

func (c ClientHelloSnapshot) GetTime() time.Time {
	return time.Unix(int64(c.Time), 0)
}

func (c ClientHelloSnapshot) GetRandom() []byte {
	data, _ := base64.StdEncoding.DecodeString(c.Random)

	return data
}

func (c ClientHelloSnapshot) GetSessionID() []byte {
	data, _ := base64.StdEncoding.DecodeString(c.SessionID)

	return data
}

func (c ClientHelloSnapshot) GetHost() string {
	return c.Host
}

func (c ClientHelloSnapshot) GetCipherSuite() uint16 {
	return uint16(c.CipherSuite)
}

func (c ClientHelloSnapshot) GetFull() []byte {
	data, _ := base64.StdEncoding.DecodeString(c.Full)

	return data
}

type ClientHelloTestSuite struct {
	suite.Suite

	secret mtglib.Secret
}

func (suite *ClientHelloTestSuite) SetupSuite() {
	parsed, err := mtglib.ParseSecret("ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d")
	if err != nil {
		panic(err)
	}

	suite.secret = parsed
}

func (suite *ClientHelloTestSuite) TestEmptyHandshake() {
	_, err := faketls.ParseClientHello(suite.secret.Key[:], nil)
	suite.Error(err)
}

func (suite *ClientHelloTestSuite) TestIncorrectHandshakeType() {
	data := make([]byte, 1024)
	data[0] = 0x02

	_, err := faketls.ParseClientHello(suite.secret.Key[:], data)
	suite.Error(err)
}

func (suite *ClientHelloTestSuite) TestIncorrectLength() {
	data := make([]byte, 1024)
	data[0] = 0x01
	data[1] = 0xff
	data[2] = 0xff

	_, err := faketls.ParseClientHello(suite.secret.Key[:], data)
	suite.Error(err)
}

func (suite *ClientHelloTestSuite) TestSnapshotOk() {
	files, err := os.ReadDir("testdata")
	suite.NoError(err)

	testData := []string{}

	for _, v := range files {
		if strings.HasPrefix(v.Name(), "client-hello-ok") {
			testData = append(testData, v.Name())
		}
	}

	for _, name := range testData {
		path := filepath.Join("testdata", name)

		suite.T().Run(name, func(t *testing.T) {
			fileData, err := os.ReadFile(path)
			assert.NoError(t, err)

			snapshot := &ClientHelloSnapshot{}
			assert.NoError(t, json.Unmarshal(fileData, snapshot))

			hello, err := faketls.ParseClientHello(suite.secret.Key[:], snapshot.GetFull())
			assert.NoError(t, err)
			assert.WithinDuration(t, snapshot.GetTime(), hello.Time, time.Second)
			assert.Equal(t, snapshot.GetRandom(), hello.Random[:])
			assert.Equal(t, snapshot.GetSessionID(), hello.SessionID)
			assert.Equal(t, snapshot.GetHost(), hello.Host)
			assert.Equal(t, snapshot.GetCipherSuite(), hello.CipherSuite)
		})
	}
}

func (suite *ClientHelloTestSuite) TestSnapshotBad() {
	files, err := os.ReadDir("testdata")
	suite.NoError(err)

	testData := []string{}

	for _, v := range files {
		if strings.HasPrefix(v.Name(), "client-hello-bad") {
			testData = append(testData, v.Name())
		}
	}

	for _, name := range testData {
		path := filepath.Join("testdata", name)

		suite.T().Run(name, func(t *testing.T) {
			fileData, err := os.ReadFile(path)
			assert.NoError(t, err)

			snapshot := &ClientHelloSnapshot{}
			assert.NoError(t, json.Unmarshal(fileData, snapshot))

			_, err = faketls.ParseClientHello(suite.secret.Key[:], snapshot.GetFull())
			assert.Error(t, err)
		})
	}
}

func (suite *ClientHelloTestSuite) TestValidateHostname() {
	hello := faketls.ClientHello{
		Time: time.Now(),
	}
	suite.NoError(hello.Valid("hostname", time.Second))

	hello.Host = "hostname"
	suite.Error(hello.Valid("hostname2", time.Second))
	suite.NoError(hello.Valid("hostname", time.Second))
}

func (suite *ClientHelloTestSuite) TestValidateTime() {
	testData := []time.Duration{
		-2 * time.Second,
		2 * time.Second,
	}

	for _, v := range testData {
		value := v

		suite.T().Run(value.String(), func(t *testing.T) {
			hello := faketls.ClientHello{
				Host: "hostname",
				Time: time.Now().Add(value),
			}
			suite.Error(hello.Valid("hostname", 500*time.Millisecond))
			suite.Error(hello.Valid("hostname", time.Second))
			suite.NoError(hello.Valid("hostname", 3*time.Second))
		})
	}
}

func TestClientHello(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ClientHelloTestSuite{})
}
