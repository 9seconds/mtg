package dc

import (
	"fmt"

	"github.com/9seconds/mtg/v2/mtglib/internal/obfuscation"
)

type Addr struct {
	Network    string
	Address    string
	Obfuscator obfuscation.Obfuscator
}

func (d Addr) String() string {
	return fmt.Sprintf("addr=%s, secret=%v", d.Address, d.Obfuscator.Secret)
}
