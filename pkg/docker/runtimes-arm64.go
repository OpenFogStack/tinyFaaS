//go:build arm64
// +build arm64

package docker

import "embed"

//go:embed runtimes-arm64
var runtimes embed.FS

const runtimesDir = "runtimes-arm64"
