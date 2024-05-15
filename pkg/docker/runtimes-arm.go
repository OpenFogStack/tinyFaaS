//go:build arm
// +build arm

package docker

import "embed"

//go:embed runtimes-arm
var runtimes embed.FS

const runtimesDir = "runtimes-arm"
