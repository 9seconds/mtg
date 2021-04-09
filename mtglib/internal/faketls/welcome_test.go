package faketls_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"math/rand"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/mtglib/internal/faketls"
	"github.com/9seconds/mtg/v2/mtglib/internal/faketls/record"
	"github.com/stretchr/testify/suite"
)

type WelcomeTestSuite struct {
	suite.Suite

	h      *faketls.ClientHello
	buf    *bytes.Buffer
	secret mtglib.Secret
}

func (suite *WelcomeTestSuite) SetupTest() {
	suite.h = &faketls.ClientHello{
		Time:        time.Now(),
		Host:        "google.com",
		CipherSuite: 4867,
		SessionID:   make([]byte, 32),
	}

	_, err := rand.Read(suite.h.SessionID)
	suite.NoError(err)

	_, err = rand.Read(suite.h.Random[:])
	suite.NoError(err)

	suite.buf = &bytes.Buffer{}

	suite.secret = mtglib.GenerateSecret("google.com")
}

func (suite *WelcomeTestSuite) TestOk() {
	suite.NoError(faketls.SendWelcomePacket(suite.buf, suite.secret.Key[:], *suite.h))

	welcomePacket := []byte{}
	welcomePacket = append(welcomePacket, suite.buf.Bytes()...)

	rec := record.AcquireRecord()
	defer record.ReleaseRecord(rec)

	suite.NoError(rec.Read(suite.buf))
	suite.Equal(record.TypeHandshake, rec.Type)
	suite.Equal(record.Version12, rec.Version)

	suite.NoError(rec.Read(suite.buf))
	suite.Equal(record.TypeChangeCipherSpec, rec.Type)
	suite.Equal(record.Version12, rec.Version)

	suite.NoError(rec.Read(suite.buf))
	suite.Equal(record.TypeApplicationData, rec.Type)
	suite.Equal(record.Version12, rec.Version)
	suite.Empty(suite.buf.Bytes())

	random := make([]byte, 32)
	copy(random, welcomePacket[11:])

	empty := make([]byte, 32)
	copy(welcomePacket[11:], empty)

	mac := hmac.New(sha256.New, suite.secret.Key[:])
	mac.Write(suite.h.Random[:])
	mac.Write(welcomePacket)

	suite.Equal(random, mac.Sum(nil))
}

func TestWelcome(t *testing.T) {
	t.Parallel()
	suite.Run(t, &WelcomeTestSuite{})
}
