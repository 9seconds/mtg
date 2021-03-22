package obfuscated2_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
