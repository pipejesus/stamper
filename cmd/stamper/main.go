// Command stamper stamps out Lando environments and installs agent skills.
package main

import (
	"os"
	"runtime/debug"

	"github.com/pipejesus/stamper/internal/cli"
)

// Set with -ldflags "-X main.version=… -X main.commit=…".
var (
	version = ""
	commit  = ""
)

func main() {
	switch {
	case version != "":
		cli.Version = version
	default:
		// `go install …@vX.Y.Z` records the module version in the binary.
		if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			cli.Version = bi.Main.Version
		}
	}
	cli.Commit = commit
	os.Exit(cli.Execute())
}
