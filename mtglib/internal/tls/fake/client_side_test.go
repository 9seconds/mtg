package fake_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls/fake"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	TolerateTime = 365 * 30 * 24 * time.Hour
)

type parseClientHelloConnMock struct {
	testlib.EssentialsConnMock

	readBuf *bytes.Buffer
}

func (m *parseClientHelloConnMock) Read(p []byte) (int, error) {
	return m.readBuf.Read(p)
}

type ParseClientHelloTestSuite struct {
	suite.Suite

	secret   mtglib.Secret
	readBuf  *bytes.Buffer
	connMock *parseClientHelloConnMock
}

func (suite *ParseClientHelloTestSuite) SetupSuite() {
	parsed, err := mtglib.ParseSecret("ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d")
	require.NoError(suite.T(), err)

	suite.secret = parsed
}

func (suite *ParseClientHelloTestSuite) SetupTest() {
	suite.readBuf = &bytes.Buffer{}
	suite.connMock = &parseClientHelloConnMock{
		readBuf: suite.readBuf,
	}

	suite.connMock.
		On("SetReadDeadline", mock.AnythingOfType("time.Time")).
		Twice().
		Return(nil)
}

func (suite *ParseClientHelloTestSuite) TearDownTest() {
	suite.connMock.AssertExpectations(suite.T())
}

type ParseClientHello_TLSHeaderTestSuite struct {
	ParseClientHelloTestSuite
}

func (suite *ParseClientHello_TLSHeaderTestSuite) TestEmpty() {
	suite.connMock.ExpectedCalls = []*mock.Call{}
	suite.connMock.
		On("SetReadDeadline", mock.AnythingOfType("time.Time")).
		Once().
		Return(errors.New("fail"))

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "fail")
}

func (suite *ParseClientHello_TLSHeaderTestSuite) TestNothing() {
	suite.connMock.ExpectedCalls = []*mock.Call{}
	suite.connMock.
		On("SetReadDeadline", mock.AnythingOfType("time.Time")).
		Twice().
		Return(nil)

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorIs(err, io.EOF)
}

func (suite *ParseClientHello_TLSHeaderTestSuite) TestUnknownRecord() {
	suite.readBuf.Write([]byte{
		10,
		3, 3,
		0, 0,
	})
	suite.readBuf.WriteByte(10)

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "unexpected record type 0xa")
}

func (suite *ParseClientHello_TLSHeaderTestSuite) TestUnknownProtocolVersion() {
	suite.readBuf.Write([]byte{
		tls.TypeHandshake,
		3, 3,
		0, 0,
	})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "unexpected protocol version")
}

func (suite *ParseClientHello_TLSHeaderTestSuite) TestCannotReadRestOfRecord() {
	suite.readBuf.Write([]byte{
		tls.TypeHandshake,
		3, 1,
		0, 10,
	})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorIs(err, io.EOF)
}

type ParseClientHelloHandshakeTestSuite struct {
	ParseClientHelloTestSuite
}

func (suite *ParseClientHelloHandshakeTestSuite) SetupTest() {
	suite.ParseClientHelloTestSuite.SetupTest()

	suite.readBuf.Write([]byte{
		tls.TypeHandshake,
		3, 1,
		0,
	})
}

func (suite *ParseClientHelloHandshakeTestSuite) TestCannotReadHeader() {
	suite.readBuf.Write([]byte{
		1,
		10,
	})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read handshake header")
}

func (suite *ParseClientHelloHandshakeTestSuite) TestIncorrectHandshakeType() {
	suite.readBuf.Write([]byte{
		4,
		10, 0, 0, 0,
	})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "incorrect handshake type")
}

func (suite *ParseClientHelloHandshakeTestSuite) TestCannotReadHandshake() {
	suite.readBuf.Write([]byte{
		4 + 3,
		10, 0, 0, 0,
	})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorIs(err, io.EOF)
}

type ParseClientHelloHandshakeBodyTestSuite struct {
	ParseClientHelloTestSuite
}

func (suite *ParseClientHelloHandshakeBodyTestSuite) SetupTest() {
	suite.ParseClientHelloTestSuite.SetupTest()

	suite.readBuf.Write([]byte{
		tls.TypeHandshake,
		3, 1,
		0,
	})
}

func (suite *ParseClientHelloHandshakeBodyTestSuite) writeBody(body []byte) {
	suite.readBuf.WriteByte(byte(4 + len(body)))
	suite.readBuf.Write([]byte{
		fake.TypeHandshakeClient,
		0, 0, byte(len(body)),
	})
	suite.readBuf.Write(body)
}

func (suite *ParseClientHelloHandshakeBodyTestSuite) TestCannotReadVersion() {
	suite.writeBody(nil)

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read client version")
}

func (suite *ParseClientHelloHandshakeBodyTestSuite) TestCannotReadRandom() {
	suite.writeBody([]byte{3, 3})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read client random")
}

func (suite *ParseClientHelloHandshakeBodyTestSuite) TestCannotReadSessionIDLength() {
	body := make([]byte, 2+fake.RandomLen)

	suite.writeBody(body)

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read session ID length")
}

func (suite *ParseClientHelloHandshakeBodyTestSuite) TestCannotReadSessionID() {
	body := make([]byte, 2+fake.RandomLen+1)
	body[2+fake.RandomLen] = 32

	suite.writeBody(body)

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read session id")
}

func (suite *ParseClientHelloHandshakeBodyTestSuite) TestCannotReadCipherSuiteLength() {
	body := make([]byte, 2+fake.RandomLen+1)

	suite.writeBody(body)

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read cipher suite length")
}

func (suite *ParseClientHelloHandshakeBodyTestSuite) TestCannotReadFirstCipherSuite() {
	body := make([]byte, 2+fake.RandomLen+1+2)

	suite.writeBody(body)

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read first cipher suite")
}

func (suite *ParseClientHelloHandshakeBodyTestSuite) TestCannotSkipRemainingCipherSuites() {
	body := make([]byte, 2+fake.RandomLen+1+2+2)
	binary.BigEndian.PutUint16(body[2+fake.RandomLen+1:], 4)

	suite.writeBody(body)

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot skip remaining cipher suites")
}

func (suite *ParseClientHelloHandshakeBodyTestSuite) TestCannotReadCompressionMethodsLength() {
	body := make([]byte, 2+fake.RandomLen+1+2+2)
	binary.BigEndian.PutUint16(body[2+fake.RandomLen+1:], 2)

	suite.writeBody(body)

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read compression methods length")
}

func (suite *ParseClientHelloHandshakeBodyTestSuite) TestCannotSkipCompressionMethods() {
	body := make([]byte, 2+fake.RandomLen+1+2+2+1)
	binary.BigEndian.PutUint16(body[2+fake.RandomLen+1:], 2)
	body[2+fake.RandomLen+1+2+2] = 1

	suite.writeBody(body)

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot skip compression methods")
}

type ParseClientHelloSNITestSuite struct {
	ParseClientHelloTestSuite
}

func (suite *ParseClientHelloSNITestSuite) SetupTest() {
	suite.ParseClientHelloTestSuite.SetupTest()

	suite.readBuf.Write([]byte{
		tls.TypeHandshake,
		3, 1,
		0,
	})
}

func (suite *ParseClientHelloSNITestSuite) writeExtensions(extensions []byte) {
	handshakeBodyLen := 41 + len(extensions)

	suite.readBuf.WriteByte(byte(4 + handshakeBodyLen))
	suite.readBuf.Write([]byte{
		fake.TypeHandshakeClient,
		0, 0, byte(handshakeBodyLen),
	})

	// version(2) + random(32) + sessionIDLen(1) + cipherSuiteLen(2) +
	// cipherSuite(2) + compressionLen(1) + compression(1) = 41
	body := make([]byte, 41)
	binary.BigEndian.PutUint16(body[35:], 2)
	body[39] = 1

	suite.readBuf.Write(body)
	suite.readBuf.Write(extensions)
}

func (suite *ParseClientHelloSNITestSuite) TestCannotReadExtensionsLength() {
	suite.writeExtensions(nil)

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read length of TLS extensions")
}

func (suite *ParseClientHelloSNITestSuite) TestCannotReadExtensions() {
	suite.writeExtensions([]byte{0, 10})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read extensions")
}

func (suite *ParseClientHelloSNITestSuite) TestCannotReadExtensionType() {
	suite.writeExtensions([]byte{0, 1, 0xAB})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read extension type")
}

func (suite *ParseClientHelloSNITestSuite) TestCannotReadExtensionLength() {
	suite.writeExtensions([]byte{0, 2, 0xFF, 0xFF})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "length:")
}

func (suite *ParseClientHelloSNITestSuite) TestCannotReadExtensionData() {
	suite.writeExtensions([]byte{0, 4, 0xFF, 0xFF, 0, 5})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "data: len")
}

func (suite *ParseClientHelloSNITestSuite) TestCannotReadSNIRecordLength() {
	suite.writeExtensions([]byte{0, 5, 0, 0, 0, 1, 0xAB})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read the length of the SNI record")
}

func (suite *ParseClientHelloSNITestSuite) TestCannotReadSNIListType() {
	suite.writeExtensions([]byte{0, 6, 0, 0, 0, 2, 0, 1})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "cannot read SNI list type")
}

func (suite *ParseClientHelloSNITestSuite) TestIncorrectSNIListType() {
	suite.writeExtensions([]byte{0, 7, 0, 0, 0, 3, 0, 1, 5})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "incorrect SNI list type")
}

func (suite *ParseClientHelloSNITestSuite) TestCannotReadHostnameLength() {
	suite.writeExtensions([]byte{0, 8, 0, 0, 0, 4, 0, 2, 0, 0xAB})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "incorrect length of the hostname")
}

func (suite *ParseClientHelloSNITestSuite) TestCannotReadHostname() {
	suite.writeExtensions([]byte{0, 9, 0, 0, 0, 5, 0, 3, 0, 0, 5})

	_, err := fake.ReadClientHello(suite.connMock, suite.secret.Key[:], suite.secret.Host, TolerateTime)
	suite.ErrorContains(err, "incorrect length of SNI hostname")
}

func TestParseClientHelloTLSHeader(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ParseClientHello_TLSHeaderTestSuite{})
}

func TestParseClientHelloHandshake(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ParseClientHelloHandshakeTestSuite{})
}

func TestParseClientHelloHandshakeBody(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ParseClientHelloHandshakeBodyTestSuite{})
}

func TestParseClientHelloSNI(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ParseClientHelloSNITestSuite{})
}

// --- ReadClientHelloMulti tests ---

type ReadClientHelloMultiTestSuite struct {
	suite.Suite

	secret mtglib.Secret
}

func (suite *ReadClientHelloMultiTestSuite) SetupSuite() {
	parsed, err := mtglib.ParseSecret(
		"ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d",
	)
	require.NoError(suite.T(), err)

	suite.secret = parsed
}

func (suite *ReadClientHelloMultiTestSuite) loadSnapshot(name string) []byte {
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(suite.T(), err)

	snapshot := &clientHelloSnapshot{}
	require.NoError(suite.T(), json.Unmarshal(data, snapshot))

	return snapshot.GetFull()
}

func (suite *ReadClientHelloMultiTestSuite) makeConn(data []byte) *parseClientHelloConnMock {
	readBuf := &bytes.Buffer{}
	readBuf.Write(data)

	connMock := &parseClientHelloConnMock{
		readBuf: readBuf,
	}

	connMock.
		On("SetReadDeadline", mock.AnythingOfType("time.Time")).
		Twice().
		Return(nil)

	return connMock
}

func (suite *ReadClientHelloMultiTestSuite) TestMatchesCorrectSecretAtIndex0() {
	payload := suite.loadSnapshot("client-hello-ok-19dfe38384b9884b.json")
	connMock := suite.makeConn(payload)
	defer connMock.AssertExpectations(suite.T())

	wrongSecret := mtglib.GenerateSecret("storage.googleapis.com")

	result, err := fake.ReadClientHelloMulti(
		connMock,
		[][]byte{suite.secret.Key[:], wrongSecret.Key[:]},
		suite.secret.Host,
		TolerateTime,
	)
	suite.NoError(err)
	suite.Equal(0, result.MatchedIndex)
	suite.NotNil(result.Hello)
}

func (suite *ReadClientHelloMultiTestSuite) TestMatchesCorrectSecretAtIndex1() {
	payload := suite.loadSnapshot("client-hello-ok-19dfe38384b9884b.json")
	connMock := suite.makeConn(payload)
	defer connMock.AssertExpectations(suite.T())

	wrongSecret := mtglib.GenerateSecret("storage.googleapis.com")

	result, err := fake.ReadClientHelloMulti(
		connMock,
		[][]byte{wrongSecret.Key[:], suite.secret.Key[:]},
		suite.secret.Host,
		TolerateTime,
	)
	suite.NoError(err)
	suite.Equal(1, result.MatchedIndex)
	suite.NotNil(result.Hello)
}

func (suite *ReadClientHelloMultiTestSuite) TestMatchesCorrectSecretAtIndex2() {
	payload := suite.loadSnapshot("client-hello-ok-19dfe38384b9884b.json")
	connMock := suite.makeConn(payload)
	defer connMock.AssertExpectations(suite.T())

	wrong1 := mtglib.GenerateSecret("storage.googleapis.com")
	wrong2 := mtglib.GenerateSecret("storage.googleapis.com")

	result, err := fake.ReadClientHelloMulti(
		connMock,
		[][]byte{wrong1.Key[:], wrong2.Key[:], suite.secret.Key[:]},
		suite.secret.Host,
		TolerateTime,
	)
	suite.NoError(err)
	suite.Equal(2, result.MatchedIndex)
	suite.NotNil(result.Hello)
}

func (suite *ReadClientHelloMultiTestSuite) TestNoMatchReturnsBadDigest() {
	payload := suite.loadSnapshot("client-hello-ok-19dfe38384b9884b.json")
	connMock := suite.makeConn(payload)
	defer connMock.AssertExpectations(suite.T())

	wrong1 := mtglib.GenerateSecret("storage.googleapis.com")
	wrong2 := mtglib.GenerateSecret("storage.googleapis.com")

	_, err := fake.ReadClientHelloMulti(
		connMock,
		[][]byte{wrong1.Key[:], wrong2.Key[:]},
		suite.secret.Host,
		TolerateTime,
	)
	suite.ErrorIs(err, fake.ErrBadDigest)
}

func (suite *ReadClientHelloMultiTestSuite) TestBadSnapshotReturnsBadDigest() {
	payload := suite.loadSnapshot("client-hello-bad-fa2e46cdb33e2a1b.json")
	connMock := suite.makeConn(payload)
	defer connMock.AssertExpectations(suite.T())

	_, err := fake.ReadClientHelloMulti(
		connMock,
		[][]byte{suite.secret.Key[:]},
		suite.secret.Host,
		TolerateTime,
	)
	suite.ErrorIs(err, fake.ErrBadDigest)
}

func TestReadClientHelloMulti(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ReadClientHelloMultiTestSuite{})
}
