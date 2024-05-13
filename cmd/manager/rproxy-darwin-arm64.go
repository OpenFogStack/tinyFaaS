//go:build darwin && arm64
// +build darwin,arm64

package main

import _ "embed"

//go:embed rproxy-darwin-arm64.bin
var RProxyBin []byte
