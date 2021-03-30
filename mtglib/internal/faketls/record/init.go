package record

import "fmt"

const TLSMaxRecordSize = 65535 // max uint16

type Type uint8

const (
	TypeChangeCipherSpec Type = 0x14
	TypeHandshake        Type = 0x16
	TypeApplicationData  Type = 0x17
)

func (t Type) String() string {
	switch t {
	case TypeChangeCipherSpec:
		return "changeCipher(0x14)"
	case TypeHandshake:
		return "handshake(0x16)"
	case TypeApplicationData:
		return "applicationData(0x17)"
	}

	return fmt.Sprintf("unknown(%#x)", byte(t))
}

func (t Type) Valid() error {
	switch t {
	case TypeChangeCipherSpec, TypeHandshake, TypeApplicationData:
		return nil
	}

	return fmt.Errorf("unknown type %#x", byte(t))
}

type Version uint16

const (
	Version10 Version = 769 // 0x03 0x01
	Version11 Version = 770 // 0x03 0x02
	Version12 Version = 771 // 0x03 0x03
	Version13 Version = 772 // 0x03 0x04
)

func (v Version) String() string {
	switch v {
	case Version10:
		return "tls1.0"
	case Version11:
		return "tls1.1"
	case Version12:
		return "tls1.2"
	case Version13:
		return "tls1.3"
	}

	return fmt.Sprintf("tls?(%d)", uint16(v))
}

func (v Version) Valid() error {
	switch v {
	case Version10, Version11, Version12, Version13:
		return nil
	}

	return fmt.Errorf("unknown version %d", uint16(v))
}
