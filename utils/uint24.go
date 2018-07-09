package utils

// Uint24 is a replacement for the absent Go uint24 data type.
// This data type is little endian.
type Uint24 [3]byte

// ToUint24 converts number to Uint24.
func ToUint24(number uint32) Uint24 {
	return Uint24{byte(number), byte(number >> 8), byte(number >> 16)}
}

// FromUint24 converts Uint24 to number.
func FromUint24(number Uint24) uint32 {
	return uint32(number[0]) + (uint32(number[1]) << 8) + (uint32(number[2]) << 16)
}
