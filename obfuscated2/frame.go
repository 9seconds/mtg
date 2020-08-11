package obfuscated2

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

// [frameOffsetFirst:frameOffsetKey:frameOffsetIV:frameOffsetMagic:frameOffsetDC:frameOffsetEnd].
type Frame struct {
	data [frameLen]byte
}

func (f *Frame) Bytes() []byte {
	return f.data[:]
}

func (f *Frame) Key() []byte {
	return f.data[frameOffsetFirst:frameOffsetKey]
}

func (f *Frame) IV() []byte {
	return f.data[frameOffsetKey:frameOffsetIV]
}

func (f *Frame) Magic() []byte {
	return f.data[frameOffsetIV:frameOffsetMagic]
}

func (f *Frame) DC() []byte {
	return f.data[frameOffsetMagic:frameOffsetDC]
}

func (f *Frame) Unique() []byte {
	return f.data[frameOffsetFirst:frameOffsetDC]
}

func (f *Frame) Invert() (nf Frame) {
	nf = *f
	for i := 0; i < frameLenKey+frameLenIV; i++ {
		nf.data[frameOffsetFirst+i] = f.data[frameOffsetIV-1-i]
	}

	return
}
