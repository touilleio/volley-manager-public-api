package buildinfo

import "runtime"

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	OSArch    = runtime.GOOS + "/" + runtime.GOARCH
)
