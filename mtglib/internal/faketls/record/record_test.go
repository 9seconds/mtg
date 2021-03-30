package record_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/9seconds/mtg/v2/mtglib/internal/faketls/record"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RecordTestSnapshot struct {
	Type    int    `json:"type"`
	Version int    `json:"version"`
	Payload string `json:"payload"`
	Record  string `json:"record"`
}

func (r RecordTestSnapshot) RecordBytes() []byte {
	data, _ := base64.StdEncoding.DecodeString(r.Record)

	return data
}

func (r RecordTestSnapshot) PayloadBytes() []byte {
	data, _ := base64.StdEncoding.DecodeString(r.Payload)

	return data
}

type RecordTestSuite struct {
	suite.Suite

	r   *record.Record
	buf *bytes.Buffer
}

func (suite *RecordTestSuite) SetupTest() {
	suite.r = record.AcquireRecord()
	suite.buf = &bytes.Buffer{}
}

func (suite *RecordTestSuite) TearDownTest() {
	record.ReleaseRecord(suite.r)
	suite.buf.Reset()
}

func (suite *RecordTestSuite) TestIdempotent() {
	suite.r.Type = record.TypeApplicationData
	suite.r.Version = record.Version13

	suite.r.Payload.Write([]byte{1, 2, 3})
	suite.NoError(suite.r.Dump(suite.buf))

	suite.r.Reset()
	suite.NoError(suite.r.Read(suite.buf))

	suite.Equal(0, suite.buf.Len())
	suite.Equal(record.TypeApplicationData, suite.r.Type)
	suite.Equal(record.Version13, suite.r.Version)
	suite.Equal([]byte{1, 2, 3}, suite.r.Payload.Bytes())
}

func (suite *RecordTestSuite) TestString() {
	_ = suite.r.String()
}

func (suite *RecordTestSuite) TestSnapshot() {
	files, err := os.ReadDir("testdata")
	suite.NoError(err)

	testData := map[string]string{}

	for _, f := range files {
		testData[f.Name()] = filepath.Join("testdata", f.Name())
	}

	for name, pathV := range testData {
		path := pathV

		suite.T().Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			assert.NoError(t, err)

			snapshot := &RecordTestSnapshot{}
			assert.NoError(t, json.Unmarshal(data, snapshot))

			rec := record.AcquireRecord()
			defer record.ReleaseRecord(rec)

			assert.NoError(t, rec.Read(bytes.NewReader(snapshot.RecordBytes())))
			assert.Equal(t, snapshot.Type, int(rec.Type))
			assert.Equal(t, snapshot.Version, int(rec.Version))
			assert.Equal(t, snapshot.PayloadBytes(), rec.Payload.Bytes())

			buf := &bytes.Buffer{}
			assert.NoError(t, rec.Dump(buf))
			assert.Equal(t, snapshot.RecordBytes(), buf.Bytes())
		})
	}
}

func TestRecord(t *testing.T) {
	t.Parallel()
	suite.Run(t, &RecordTestSuite{})
}
