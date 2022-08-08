package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"runtime/debug"
	"sort"
	"strconv"
	"time"
)

var version = "dev" // has to be set by ldflags

const (
	buildInfoModuleStart byte = iota
	buildInfoModuleFinish
	buildInfoModuleDelimeter
)

func getVersion() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}

	date := time.Now()
	commit := ""
	goVersion := buildInfo.GoVersion
	dirtySuffix := ""

	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.time":
			date, _ = time.Parse(time.RFC3339, setting.Value)
		case "vcs.revision":
			commit = setting.Value
		case "vcs.modified":
			if dirty, _ := strconv.ParseBool(setting.Value); dirty {
				dirtySuffix = " [dirty]"
			}
		}
	}

	hasher := sha256.New()

	checksumModule := func(mod *debug.Module) {
		hasher.Write([]byte{buildInfoModuleStart})

		io.WriteString(hasher, mod.Path) //nolint: errcheck
		hasher.Write([]byte{buildInfoModuleDelimeter})

		io.WriteString(hasher, mod.Version) //nolint: errcheck
		hasher.Write([]byte{buildInfoModuleDelimeter})

		io.WriteString(hasher, mod.Sum) //nolint: errcheck

		hasher.Write([]byte{buildInfoModuleFinish})
	}

	io.WriteString(hasher, buildInfo.Path) //nolint: errcheck

	binary.Write(hasher, binary.LittleEndian, uint64(1+len(buildInfo.Deps))) //nolint: errcheck

	sort.Slice(buildInfo.Deps, func(i, j int) bool {
		return buildInfo.Deps[i].Path > buildInfo.Deps[j].Path
	})

	checksumModule(&buildInfo.Main)

	for _, module := range buildInfo.Deps {
		checksumModule(module)
	}

	return fmt.Sprintf("%s (%s: %s on %s%s, modules checksum %s)",
		version,
		goVersion,
		date.Format(time.RFC3339),
		commit,
		dirtySuffix,
		base64.StdEncoding.EncodeToString(hasher.Sum(nil)))
}
