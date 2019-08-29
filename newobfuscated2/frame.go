package newobfuscated2

const (
	frameLenKey   = 32
	frameLenIV    = 16
	frameLenMagic = 4
	frameLenDC    = 2

	frameOffsetFirst = 8
	frameOffsetKey   = frameOffsetFirst + frameLenKey
	frameOffsetIV    = frameOffsetKey + frameLenIV
	frameOffsetMagic = frameOffsetIV + frameLenMagic
	frameOffsetDC    = frameOffsetMagic + frameLenDC

	frameLen = 64
)

// [frameOffsetFirst:frameOffsetKey:frameOffsetIV:frameOffsetMagic:frameOffsetDC:frameOffsetEnd]
type frame struct {
	data [frameLen]byte
}

func (f *frame) bytes() []byte {
	return f.data[:]
}

func (f *frame) key() []byte {
	return f.data[frameOffsetFirst:frameOffsetKey]
}

func (f *frame) iv() []byte {
	return f.data[frameOffsetKey:frameOffsetIV]
}

func (f *frame) magic() []byte {
	return f.data[frameOffsetIV:frameOffsetMagic]
}

func (f *frame) dc() []byte {
	return f.data[frameOffsetMagic:frameOffsetDC]
}

func (f *frame) unique() []byte {
	return f.data[frameOffsetFirst:frameOffsetDC]
}

func (f *frame) invert() (nf frame) {
	nf = *f
	for i := 0; i < frameLenKey+frameLenIV; i++ {
		nf.data[frameOffsetFirst+i] = nf.data[frameOffsetIV-1-i]
	}

	return
}
