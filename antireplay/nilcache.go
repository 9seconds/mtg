package antireplay

type nilCache struct{}

func (n nilCache) AddObfuscated2(_ []byte)      {}
func (n nilCache) AddTLS(_ []byte)              {}
func (n nilCache) HasObfuscated2(_ []byte) bool { return false }
func (n nilCache) HasTLS(_ []byte) bool         { return false }
