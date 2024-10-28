package main

import (
	"runtime/debug"

	cmd "github.com/magicdrive/kirke/cmd/kirke"
)

var version string

func main() {
	cmd.Excecute(Version())
}

func Version() string {
	if version != "" {
		return version
	}

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		return buildInfo.Main.Version
	}
	return "unknown"

}
