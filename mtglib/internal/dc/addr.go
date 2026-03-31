package dc

import (
	"fmt"

	"github.com/dolonet/mtg-multi/mtglib/internal/obfuscation"
)

type Addr struct {
	Network    string
	Address    string
	Obfuscator obfuscation.Obfuscator
}

func (d Addr) String() string {
	return fmt.Sprintf("addr=%s, secret=%v", d.Address, d.Obfuscator.Secret)
}
