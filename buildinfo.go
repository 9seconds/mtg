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
	goVersion, date, commit, modulesChecksum, dirty := getVersionData()

	dirtySuffix := ""
	if dirty {
		dirtySuffix = " [dirty]"
	}

	return fmt.Sprintf("%s (%s: %s on %s%s, modules checksum %s)",
		version,
		goVersion,
		date.Format(time.RFC3339),
		commit,
		dirtySuffix,
		modulesChecksum)
}

func getVersionData() (goVersion string, date time.Time, commit string, modulesChecksum string, dirty bool) {
	date = time.Now()

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	goVersion = buildInfo.GoVersion

	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.time":
			date, _ = time.Parse(time.RFC3339, setting.Value)
		case "vcs.revision":
			commit = setting.Value
		case "vcs.modified":
			dirty, _ = strconv.ParseBool(setting.Value)
		}
	}

	hasher := sha256.New()
	if _, err := io.WriteString(hasher, buildInfo.Path); err != nil {
		panic(err)
	}
	binary.Write(hasher, binary.LittleEndian, uint64(1+len(buildInfo.Deps)))

	sort.Slice(buildInfo.Deps, func(i, j int) bool {
		return buildInfo.Deps[i].Path > buildInfo.Deps[j].Path
	})

	buildInfoCheckSumModule(hasher, &buildInfo.Main)
	for _, module := range buildInfo.Deps {
		buildInfoCheckSumModule(hasher, module)
	}

	modulesChecksum = base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	return
}

func buildInfoCheckSumModule(w io.Writer, module *debug.Module) {
	w.Write([]byte{buildInfoModuleStart})

	if _, err := io.WriteString(w, module.Path); err != nil {
		panic(err)
	}

	w.Write([]byte{buildInfoModuleDelimeter})

	if _, err := io.WriteString(w, module.Version); err != nil {
		panic(err)
	}

	w.Write([]byte{buildInfoModuleDelimeter})

	if _, err := io.WriteString(w, module.Sum); err != nil {
		panic(err)
	}

	w.Write([]byte{buildInfoModuleFinish})
}
