package cli

import "github.com/alecthomas/kong"

type CLI struct {
	GenerateSecret GenerateSecret   `kong:"cmd,help='Generate new proxy secret'"` // nolint: govet
	Access         Access           `kong:"cmd,help='Print access information.'"` // nolint: govet
	Version        kong.VersionFlag `kong:"help='Print version.'"`
}
