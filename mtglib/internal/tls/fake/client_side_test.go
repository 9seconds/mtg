package fake_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/dolonet/mtg-multi/internal/testlib"
	"github.com/dolonet/mtg-multi/mtglib"
	"github.com/dolonet/mtg-multi/mtglib/internal/tls"
	"github.com/dolonet/mtg-multi/mtglib/internal/tls/fake"
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

// fragmentTLSRecord splits a single TLS record into n TLS records by
// dividing the payload into roughly equal parts. Each part gets its own
// TLS record header with the same record type and version.
func fragmentTLSRecord(t testing.TB, full []byte, n int) []byte {
	t.Helper()

	recordType := full[0]
	version := full[1:3]
	payload := full[tls.SizeHeader:]

	chunkSize := len(payload) / n
	result := &bytes.Buffer{}

	for i := 0; i < n; i++ {
		start := i * chunkSize
		end := start + chunkSize

		if i == n-1 {
			end = len(payload)
		}

		chunk := payload[start:end]
		result.WriteByte(recordType)
		result.Write(version)
		require.NoError(t, binary.Write(result, binary.BigEndian, uint16(len(chunk))))
		result.Write(chunk)
	}

	return result.Bytes()
}

// splitPayloadAt creates two TLS records from a single record by splitting
// the payload at the given byte position.
func splitPayloadAt(t testing.TB, full []byte, pos int) []byte {
	t.Helper()

	payload := full[tls.SizeHeader:]
	buf := &bytes.Buffer{}

	buf.WriteByte(tls.TypeHandshake)
	buf.Write(full[1:3])
	require.NoError(t, binary.Write(buf, binary.BigEndian, uint16(pos)))
	buf.Write(payload[:pos])

	buf.WriteByte(tls.TypeHandshake)
	buf.Write(full[1:3])
	require.NoError(t, binary.Write(buf, binary.BigEndian, uint16(len(payload)-pos)))
	buf.Write(payload[pos:])

	return buf.Bytes()
}

type ParseClientHelloFragmentedTestSuite struct {
	suite.Suite

	secret   mtglib.Secret
	snapshot *clientHelloSnapshot
}

func (s *ParseClientHelloFragmentedTestSuite) SetupSuite() {
	parsed, err := mtglib.ParseSecret(
		"ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d",
	)
	require.NoError(s.T(), err)

	s.secret = parsed

	fileData, err := os.ReadFile("testdata/client-hello-ok-19dfe38384b9884b.json")
	require.NoError(s.T(), err)

	s.snapshot = &clientHelloSnapshot{}
	require.NoError(s.T(), json.Unmarshal(fileData, s.snapshot))
}

func (s *ParseClientHelloFragmentedTestSuite) makeConn(data []byte) *parseClientHelloConnMock {
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

func (s *ParseClientHelloFragmentedTestSuite) TestReassemblySuccess() {
	full := s.snapshot.GetFull()

	tests := []struct {
		name string
		data []byte
	}{
		{"two equal fragments", fragmentTLSRecord(s.T(), full, 2)},
		{"three equal fragments", fragmentTLSRecord(s.T(), full, 3)},
		{"single byte first fragment", splitPayloadAt(s.T(), full, 1)},
		{"three byte first fragment", splitPayloadAt(s.T(), full, 3)},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			connMock := s.makeConn(tt.data)
			defer connMock.AssertExpectations(s.T())

			hello, err := fake.ReadClientHello(
				connMock,
				s.secret.Key[:],
				s.secret.Host,
				TolerateTime,
			)
			s.Require().NoError(err)

			s.Equal(s.snapshot.GetRandom(), hello.Random[:])
			s.Equal(s.snapshot.GetSessionID(), hello.SessionID)
			s.Equal(uint16(s.snapshot.CipherSuite), hello.CipherSuite)
		})
	}
}

func (s *ParseClientHelloFragmentedTestSuite) TestReassemblyErrors() {
	full := s.snapshot.GetFull()
	payload := full[tls.SizeHeader:]

	tests := []struct {
		name      string
		buildData func() []byte
		errMsg    string
	}{
		{
			name: "wrong continuation record type",
			buildData: func() []byte {
				buf := &bytes.Buffer{}
				buf.WriteByte(tls.TypeHandshake)
				buf.Write(full[1:3])
				require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(10)))
				buf.Write(payload[:10])
				// Wrong type: application data instead of handshake
				buf.WriteByte(tls.TypeApplicationData)
				buf.Write(full[1:3])
				require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(len(payload)-10)))
				buf.Write(payload[10:])
				return buf.Bytes()
			},
			errMsg: "unexpected record type",
		},
		{
			name: "too many continuation records",
			buildData: func() []byte {
				// Handshake header claiming 256 bytes, but we only send 1 byte per continuation
				handshakePayload := []byte{0x01, 0x00, 0x01, 0x00}
				buf := &bytes.Buffer{}
				buf.WriteByte(tls.TypeHandshake)
				buf.Write([]byte{3, 1})
				require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(len(handshakePayload))))
				buf.Write(handshakePayload)
				for range 11 {
					buf.WriteByte(tls.TypeHandshake)
					buf.Write([]byte{3, 1})
					require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(1)))
					buf.WriteByte(0xAB)
				}
				return buf.Bytes()
			},
			errMsg: "too many fragments",
		},
		{
			name: "zero-length continuation record",
			buildData: func() []byte {
				buf := &bytes.Buffer{}
				buf.WriteByte(tls.TypeHandshake)
				buf.Write(full[1:3])
				require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(10)))
				buf.Write(payload[:10])
				// Valid header but zero-length payload
				buf.WriteByte(tls.TypeHandshake)
				buf.Write(full[1:3])
				require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(0)))
				return buf.Bytes()
			},
			errMsg: "cannot read record header",
		},
		{
			name: "wrong continuation record version",
			buildData: func() []byte {
				buf := &bytes.Buffer{}
				buf.WriteByte(tls.TypeHandshake)
				buf.Write(full[1:3])
				require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(10)))
				buf.Write(payload[:10])
				// Wrong version: 3.3 instead of 3.1
				buf.WriteByte(tls.TypeHandshake)
				buf.Write([]byte{3, 3})
				require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(len(payload)-10)))
				buf.Write(payload[10:])
				return buf.Bytes()
			},
			errMsg: "unexpected protocol version",
		},
		{
			name: "handshake message too large",
			buildData: func() []byte {
				// Handshake header claiming 0x010000 (65536) bytes — exceeds 0xFFFF limit
				handshakePayload := []byte{0x01, 0x01, 0x00, 0x00}
				buf := &bytes.Buffer{}
				buf.WriteByte(tls.TypeHandshake)
				buf.Write([]byte{3, 1})
				require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(len(handshakePayload))))
				buf.Write(handshakePayload)
				return buf.Bytes()
			},
			errMsg: "cannot read record header",
		},
		{
			name: "truncated continuation record header",
			buildData: func() []byte {
				buf := &bytes.Buffer{}
				buf.WriteByte(tls.TypeHandshake)
				buf.Write(full[1:3])
				require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(10)))
				buf.Write(payload[:10])
				// Connection ends mid-header (only 2 bytes)
				buf.WriteByte(tls.TypeHandshake)
				buf.WriteByte(3)
				return buf.Bytes()
			},
			errMsg: "cannot read record header",
		},
		{
			name: "truncated continuation record payload",
			buildData: func() []byte {
				buf := &bytes.Buffer{}
				buf.WriteByte(tls.TypeHandshake)
				buf.Write(full[1:3])
				require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(10)))
				buf.Write(payload[:10])
				// Claims 100 bytes but no payload follows
				buf.WriteByte(tls.TypeHandshake)
				buf.Write(full[1:3])
				require.NoError(s.T(), binary.Write(buf, binary.BigEndian, uint16(100)))
				return buf.Bytes()
			},
			errMsg: "EOF",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			connMock := s.makeConn(tt.buildData())
			defer connMock.AssertExpectations(s.T())

			_, err := fake.ReadClientHello(
				connMock,
				s.secret.Key[:],
				s.secret.Host,
				TolerateTime,
			)
			s.ErrorContains(err, tt.errMsg)
		})
	}
}

func TestParseClientHelloFragmented(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ParseClientHelloFragmentedTestSuite{})
}
