package buildinfo

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOSArchMatchesRuntime(t *testing.T) {
	require.Equal(t, runtime.GOOS+"/"+runtime.GOARCH, OSArch)
}

func TestDefaultsArePresent(t *testing.T) {
	require.NotEmpty(t, Version)
	require.NotEmpty(t, GitCommit)
	require.NotEmpty(t, BuildDate)
}
