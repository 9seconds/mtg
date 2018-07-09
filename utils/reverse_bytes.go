package utils

// ReverseBytes is a common slice reverser.
func ReverseBytes(data []byte) []byte {
	dataLen := len(data)
	rv := make([]byte, dataLen)

	rv[dataLen/2] = data[dataLen/2]
	for i := dataLen/2 - 1; i >= 0; i-- {
		opp := dataLen - i - 1
		rv[i], rv[opp] = data[opp], data[i]
	}

	return rv
}
