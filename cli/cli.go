package cli

import "github.com/alecthomas/kong"

type CLI struct {
	GenerateSecret GenerateSecret   `kong:"cmd,help='Generate new proxy secret'"`
	Access         Access           `kong:"cmd,help='Print access information.'"`
	Version        kong.VersionFlag `kong:"help='Print version.',short='v'"`
}
