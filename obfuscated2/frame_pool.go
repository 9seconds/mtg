package obfuscated2

import "sync"

var framePool sync.Pool

// MakeFrame returns new pointer to the handshake frame.
func MakeFrame() *Frame {
	return framePool.Get().(*Frame)
}

// ReturnFrame returns pointer to the handshake frame back to the pool.
func ReturnFrame(f *Frame) {
	framePool.Put(f)
}

func init() {
	framePool = sync.Pool{
		New: func() interface{} {
			data := make(Frame, FrameLen)
			return &data
		},
	}
}
