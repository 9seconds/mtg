package cli

import "github.com/alecthomas/kong"

type CLI struct {
	GenerateSecret GenerateSecret   `cmd help:"Generate new proxy secret"` // nolint: govet
	Access         Access           `cmd help:"Print access information."` // nolint: govet
	Version        kong.VersionFlag `help:"Print version."`
}
