package obfuscation_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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

type ObfuscatedSnapshot struct {
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
	suite.Suite

	snapshots map[string]*ObfuscatedSnapshot
}

func (s *SnapshotTestSuite) Setup(dirname, namePrefix string) {
	s.snapshots = make(map[string]*ObfuscatedSnapshot)

	files, err := os.ReadDir("testdata")
	require.NoError(s.T(), err)

	for _, v := range files {
		if !strings.HasPrefix(v.Name(), namePrefix) {
			continue
		}

		filename := filepath.Join("testdata", v.Name())

		contents, err := os.ReadFile(filename)
		require.NoError(s.T(), err)

		value := &ObfuscatedSnapshot{}
		require.NoError(s.T(), json.Unmarshal(contents, value))

		s.snapshots[v.Name()] = value
	}
}
