//go:build amd64
// +build amd64

package docker

import "embed"

//go:embed runtimes-amd64
var runtimes embed.FS

const runtimesDir = "runtimes-amd64"
