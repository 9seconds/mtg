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
	buf[0] = TypeApplicationData

	bufV := buf[SizeRecordType:]
	copy(bufV[:SizeVersion], TLSVersion[:])

	bufS := bufV[SizeVersion:]
	binary.BigEndian.PutUint16(bufS[:SizeSize], uint16(len(payload)))

	bufP := buf[SizeHeader:]
	if n := copy(bufP, payload); n != len(payload) {
		return fmt.Errorf("copied %d bytes of payload instead of %d", n, len(payload))
	}

	_, err := w.Write(buf[:SizeHeader+len(payload)])

	return err
}
