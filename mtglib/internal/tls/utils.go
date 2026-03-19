package tls

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

func ReadRecord(r io.Reader, w io.Writer) (byte, int64, error) {
	buf := [SizeHeader]byte{}

	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, 0, err
	}

	pVer := buf[SizeRecordType:]
	pLen := pVer[SizeVersion:]

	if !bytes.Equal(TLSVersion[:], pVer[:SizeVersion]) {
		return 0, 0, fmt.Errorf("incorrect tls version %v", pVer)
	}

	length := int64(binary.BigEndian.Uint16(pLen[:SizeSize]))
	_, err := io.CopyN(w, r, length)

	return buf[0], length, err
}

func WriteRecord(w io.Writer, payload []byte) error {
	buf := [MaxRecordSize]byte{}
	copy(buf[SizeHeader:], payload)

	return WriteRecordInPlace(w, buf[:], len(payload))
}

func WriteRecordInPlace(w io.Writer, buf []byte, payloadLen int) error {
	if payloadLen > MaxRecordPayloadSize {
		return fmt.Errorf("payload %d exceeds max %d", payloadLen, MaxRecordPayloadSize)
	}

	buf[0] = TypeApplicationData
	copy(buf[SizeRecordType:SizeRecordType+SizeVersion], TLSVersion[:])
	binary.BigEndian.PutUint16(
		buf[SizeRecordType+SizeVersion:SizeRecordType+SizeVersion+SizeSize],
		uint16(payloadLen),
	)

	_, err := w.Write(buf[:SizeHeader+payloadLen])

	return err
}
