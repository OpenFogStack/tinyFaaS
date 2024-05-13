//go:build linux && arm64
// +build linux,arm64

package main

import _ "embed"

//go:embed rproxy-linux-arm64.bin
var RProxyBin []byte
