package obfuscated2_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/mtglib/internal/obfuscated2"
	"github.com/stretchr/testify/require"
)

type snapshotBytes struct {
	data []byte
}

func (s snapshotBytes) MarshalText() ([]byte, error) {
	if len(s.data) == 0 {
		return nil, nil
	}

	return []byte(base64.RawStdEncoding.EncodeToString(s.data)), nil
}

func (s *snapshotBytes) UnmarshalText(data []byte) error {
	val, err := base64.RawStdEncoding.DecodeString(string(data))
	if err != nil {
		return fmt.Errorf("cannot unmarshal %v: %w", len(val), err)
	}

	s.data = val

	return nil
}

type Obfuscated2Snapshot struct {
	Secret    snapshotBytes `json:"secret"`
	Frame     snapshotBytes `json:"frame"`
	DC        int16         `json:"dc"`
	Encrypted struct {
		Text   snapshotBytes `json:"text"`
		Cipher snapshotBytes `json:"cipher"`
	} `json:"encrypted"`
	Decrypted struct {
		Text   snapshotBytes `json:"text"`
		Cipher snapshotBytes `json:"cipher"`
	} `json:"decrypted"`
}

type SnapshotTestSuite struct {
	snapshots map[string]*Obfuscated2Snapshot
}

type ServerHandshakeTestData struct {
	connMock *testlib.EssentialsConnMock

	proxyConn obfuscated2.Conn
	encryptor cipher.Stream
	decryptor cipher.Stream
}

func (suite *SnapshotTestSuite) IngestSnapshots(dirname, namePrefix string) error {
	suite.snapshots = map[string]*Obfuscated2Snapshot{}

	files, err := os.ReadDir(filepath.Join("testdata", dirname))
	if err != nil {
		return fmt.Errorf("cannot ingest snapshots: %w", err)
	}

	for _, v := range files {
		if !strings.HasPrefix(v.Name(), namePrefix) {
			continue
		}

		filename := filepath.Join("testdata", dirname, v.Name())

		contents, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", filename, err)
		}

		value := &Obfuscated2Snapshot{}

		if err := json.Unmarshal(contents, value); err != nil {
			return fmt.Errorf("cannot unmarshal %s: %w", filename, err)
		}

		suite.snapshots[v.Name()] = value
	}

	return nil
}

func NewServerHandshakeTestData(t *testing.T) ServerHandshakeTestData {
	buf := &bytes.Buffer{}
	connMock := &testlib.EssentialsConnMock{}

	handshakeEnc, handshakeDec, err := obfuscated2.ServerHandshake(buf)
	require.NoError(t, err)

	serverEncrypted := buf.Bytes()
	decBlock, _ := aes.NewCipher(serverEncrypted[8 : 8+32])
	decryptor := cipher.NewCTR(decBlock, serverEncrypted[8+32:8+32+16])

	serverDecrypted := make([]byte, len(serverEncrypted))
	decryptor.XORKeyStream(serverDecrypted, serverEncrypted)

	require.Equal(t, "3d3d3Q",
		base64.RawStdEncoding.EncodeToString(serverDecrypted[8+32+16:8+32+16+4]))

	serverEncryptedReverted := make([]byte, len(serverEncrypted))

	for i := 0; i < 32+16; i++ {
		serverEncryptedReverted[8+i] = serverEncrypted[8+32+16-1-i]
	}

	encBlock, _ := aes.NewCipher(serverEncryptedReverted[8 : 8+32])
	encryptor := cipher.NewCTR(encBlock, serverEncryptedReverted[8+32:8+32+16])

	return ServerHandshakeTestData{
		connMock: connMock,
		proxyConn: obfuscated2.Conn{
			Conn:      connMock,
			Encryptor: handshakeEnc,
			Decryptor: handshakeDec,
		},
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
