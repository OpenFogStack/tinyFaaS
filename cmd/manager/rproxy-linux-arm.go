//go:build linux && arm
// +build linux,arm

package main

import _ "embed"

//go:embed rproxy-linux-arm.bin
var RProxyBin []byte
