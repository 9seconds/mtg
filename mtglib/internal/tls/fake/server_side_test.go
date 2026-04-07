package fake_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/dolonet/mtg-multi/mtglib"
	"github.com/dolonet/mtg-multi/mtglib/internal/tls"
	"github.com/dolonet/mtg-multi/mtglib/internal/tls/fake"
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
	err := fake.SendServerHello(suite.buf, suite.secret.Key[:], suite.hello, fake.NoiseParams{})
	suite.NoError(err)

	var rec bytes.Buffer

	recordType, _, err := tls.ReadRecord(suite.buf, &rec)
	suite.NoError(err)
	suite.Equal(byte(tls.TypeHandshake), recordType)

	rec.Reset()

	recordType, _, err = tls.ReadRecord(suite.buf, &rec)
	suite.NoError(err)
	suite.Equal(byte(tls.TypeChangeCipherSpec), recordType)

	rec.Reset()

	recordType, length, err := tls.ReadRecord(suite.buf, &rec)
	suite.NoError(err)
	suite.Equal(byte(tls.TypeApplicationData), recordType)
	suite.GreaterOrEqual(length, int64(2500))

	suite.Empty(suite.buf.Bytes())
}

func (suite *SendServerHelloTestSuite) TestHMAC() {
	err := fake.SendServerHello(suite.buf, suite.secret.Key[:], suite.hello, fake.NoiseParams{})
	suite.NoError(err)

	packet := make([]byte, suite.buf.Len())
	copy(packet, suite.buf.Bytes())

	random := make([]byte, fake.RandomLen)
	copy(random, packet[fake.RandomOffset:])
	copy(packet[fake.RandomOffset:], make([]byte, fake.RandomLen))

	mac := hmac.New(sha256.New, suite.secret.Key[:])
	mac.Write(suite.hello.Random[:])
	mac.Write(packet)

	suite.Equal(random, mac.Sum(nil))
}

func (suite *SendServerHelloTestSuite) TestHandshakePayload() {
	err := fake.SendServerHello(suite.buf, suite.secret.Key[:], suite.hello, fake.NoiseParams{})
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
	err := fake.SendServerHello(suite.buf, suite.secret.Key[:], suite.hello, fake.NoiseParams{})
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

func (suite *SendServerHelloTestSuite) TestCalibratedNoiseSize() {
	noise := fake.NoiseParams{Mean: 6480, Jitter: 100}
	err := fake.SendServerHello(suite.buf, suite.secret.Key[:], suite.hello, noise)
	suite.NoError(err)

	var rec bytes.Buffer

	// Skip ServerHello
	_, _, err = tls.ReadRecord(suite.buf, &rec)
	suite.NoError(err)

	// Skip ChangeCipherSpec
	rec.Reset()
	_, _, err = tls.ReadRecord(suite.buf, &rec)
	suite.NoError(err)

	// Read noise ApplicationData
	rec.Reset()
	recordType, length, err := tls.ReadRecord(suite.buf, &rec)
	suite.NoError(err)
	suite.Equal(byte(tls.TypeApplicationData), recordType)

	// Should be within mean ± jitter range.
	suite.GreaterOrEqual(length, int64(noise.Mean-noise.Jitter))
	suite.LessOrEqual(length, int64(noise.Mean+noise.Jitter))
}

func TestSendServerHello(t *testing.T) {
	t.Parallel()
	suite.Run(t, &SendServerHelloTestSuite{})
}
