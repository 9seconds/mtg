package fake_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls/fake"
	"github.com/stretchr/testify/suite"
)

type SendServerHelloTestSuite struct {
	suite.Suite

	hello  *fake.ClientHello
	buf    *bytes.Buffer
	secret mtglib.Secret
}

func (suite *SendServerHelloTestSuite) SetupTest() {
	suite.hello = &fake.ClientHello{
		CipherSuite: 4867,
		SessionID:   make([]byte, 32),
	}

	_, err := rand.Read(suite.hello.SessionID)
	suite.NoError(err)

	_, err = rand.Read(suite.hello.Random[:])
	suite.NoError(err)

	suite.buf = &bytes.Buffer{}
	suite.secret = mtglib.GenerateSecret("google.com")
}

func (suite *SendServerHelloTestSuite) TestRecordStructure() {
	noise, err := fake.SendServerHello(suite.buf, suite.secret.Key[:], suite.hello)
	suite.NoError(err)

	var rec bytes.Buffer

	recordType, _, err := tls.ReadRecord(suite.buf, &rec)
	suite.NoError(err)
	suite.Equal(byte(tls.TypeHandshake), recordType)

	rec.Reset()

	recordType, _, err = tls.ReadRecord(suite.buf, &rec)
	suite.NoError(err)
	suite.Equal(byte(tls.TypeChangeCipherSpec), recordType)

	suite.Empty(suite.buf.Bytes())

	// noise is raw payload without TLS record header
	suite.Len(noise, 1369)
}

func (suite *SendServerHelloTestSuite) TestHMAC() {
	noise, err := fake.SendServerHello(suite.buf, suite.secret.Key[:], suite.hello)
	suite.NoError(err)

	packet := make([]byte, suite.buf.Len())
	copy(packet, suite.buf.Bytes())

	random := make([]byte, fake.RandomLen)
	copy(random, packet[fake.RandomOffset:])
	copy(packet[fake.RandomOffset:], make([]byte, fake.RandomLen))

	mac := hmac.New(sha256.New, suite.secret.Key[:])
	mac.Write(suite.hello.Random[:])
	mac.Write(packet)

	// HMAC is computed over the full noise TLS record (with header),
	// but SendServerHello returns noise without the header,
	// so we reconstruct the full record.
	var fullNoise bytes.Buffer
	tls.WriteRecord(&fullNoise, noise) //nolint: errcheck
	mac.Write(fullNoise.Bytes())

	suite.Equal(random, mac.Sum(nil))
}

func (suite *SendServerHelloTestSuite) TestHandshakePayload() {
	_, err := fake.SendServerHello(suite.buf, suite.secret.Key[:], suite.hello)
	suite.NoError(err)

	packet := suite.buf.Bytes()

	// TLS record header: type(1) + version(2) + length(2)
	suite.Equal(byte(tls.TypeHandshake), packet[0])
	suite.Equal([]byte{3, 3}, packet[1:3])

	// Handshake header: type(1) + uint24_length(3)
	suite.Equal(byte(fake.TypeHandshakeServer), packet[5])

	// ServerHello version
	suite.Equal([]byte{3, 3}, packet[9:11])

	// Session ID
	sessionIDOffset := fake.RandomOffset + fake.RandomLen
	suite.Equal(byte(len(suite.hello.SessionID)), packet[sessionIDOffset])
	suite.Equal(suite.hello.SessionID, packet[sessionIDOffset+1:sessionIDOffset+1+len(suite.hello.SessionID)])
}

func (suite *SendServerHelloTestSuite) TestChangeCipherSpec() {
	_, err := fake.SendServerHello(suite.buf, suite.secret.Key[:], suite.hello)
	suite.NoError(err)

	// Skip first record
	var rec bytes.Buffer

	_, _, err = tls.ReadRecord(suite.buf, &rec)
	suite.NoError(err)

	// Read ChangeCipherSpec record
	rec.Reset()

	recordType, length, err := tls.ReadRecord(suite.buf, &rec)
	suite.NoError(err)
	suite.Equal(byte(tls.TypeChangeCipherSpec), recordType)
	suite.Equal(int64(1), length)
	suite.Equal([]byte{fake.ChangeCipherValue}, rec.Bytes())
}

func TestSendServerHello(t *testing.T) {
	t.Parallel()
	suite.Run(t, &SendServerHelloTestSuite{})
}
