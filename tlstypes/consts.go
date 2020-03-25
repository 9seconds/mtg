package tlstypes

import "io"

type RecordType uint8

const (
	RecordTypeHandshake        RecordType = 0x16
	RecordTypeApplicationData  RecordType = 0x17
	RecordTypeChangeCipherSpec RecordType = 0x14
)

type HandshakeType uint8

const (
	HandshakeTypeClient HandshakeType = 0x01
	HandshakeTypeServer HandshakeType = 0x02
)

type CipherSuiteType uint8

const (
	CipherSuiteType_TLS_AES_128_GCM_SHA256       CipherSuiteType = iota // nolint: stylecheck, golint
	CipherSuiteType_TLS_AES_256_GCM_SHA384                              // nolint: stylecheck, golint
	CipherSuiteType_TLS_CHACHA20_POLY1305_SHA256                        // nolint: stylecheck, golint
)

func (c CipherSuiteType) Bytes() []byte {
	switch c {
	case CipherSuiteType_TLS_AES_128_GCM_SHA256:
		return CipherSuiteType_TLS_AES_128_GCM_SHA256_Bytes
	case CipherSuiteType_TLS_AES_256_GCM_SHA384:
		return CipherSuiteType_TLS_AES_256_GCM_SHA384_Bytes
	}

	return CipherSuiteType_TLS_CHACHA20_POLY1305_SHA256_Bytes
}

type Version uint8

func (v Version) Bytes() []byte {
	switch v {
	case Version13:
		return Version13Bytes
	case Version12:
		return Version12Bytes
	case Version11:
		return Version11Bytes
	}

	return Version10Bytes
}

const (
	VersionUnknown Version = iota
	Version10
	Version11
	Version12
	Version13
)

var (
	Version10Bytes = []byte{0x03, 0x01}
	Version11Bytes = []byte{0x03, 0x02}
	Version12Bytes = []byte{0x03, 0x03}
	Version13Bytes = []byte{0x03, 0x04}

	CipherSuiteType_TLS_AES_128_GCM_SHA256_Bytes       = []byte{0x13, 0x01} // nolint: stylecheck, golint
	CipherSuiteType_TLS_AES_256_GCM_SHA384_Bytes       = []byte{0x13, 0x02} // nolint: stylecheck, golint
	CipherSuiteType_TLS_CHACHA20_POLY1305_SHA256_Bytes = []byte{0x13, 0x03} // nolint; stylecheck, golint
)

type Byter interface {
	WriteBytes(io.Writer)
	Len() int
}

type RawBytes []byte

func (r RawBytes) WriteBytes(writer io.Writer) {
	writer.Write(r) // nolint: errcheck
}

func (r RawBytes) Len() int {
	return len(r)
}
