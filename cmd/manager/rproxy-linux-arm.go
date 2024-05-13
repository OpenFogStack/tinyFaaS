//go:build linux && armv7
// +build linux,armv7

package main

import _ "embed"

//go:embed rproxy-linux-arm.bin
var RProxyBin []byte
