//go:build linux && amd64
// +build linux,amd64

package main

import _ "embed"

//go:embed rproxy-linux-amd64.bin
var RProxyBin []byte
