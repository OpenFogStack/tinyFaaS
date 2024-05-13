PROJECT_NAME := "tinyFaaS"
PKG := "github.com/OpenFogStack/$(PROJECT_NAME)"
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v /ext/ | grep -v _test.go)
TEST_DIR := ./test

OS=$(shell go env GOOS)
ARCH=$(shell go env GOARCH)

.PHONY: all build start clean

all: build clean start

build: tinyfaas-${OS}-${ARCH}

cmd/manager/rproxy-%.bin: $(GO_FILES)
	GOOS=${OS} GOARCH=${ARCH} go build -o $@ -v $(PKG)/cmd/rproxy

tinyfaas-%: cmd/manager/rproxy-%.bin $(GO_FILES)
	GOOS=${OS} GOARCH=${ARCH} go build -o $@ -v $(PKG)/cmd/manager

start: tinyfaas-${OS}-${ARCH}
	./$<

test: build ${TEST_DIR}/test_all.py
	@python3 ${TEST_DIR}/test_all.py

clean: clean.sh
	@sh clean.sh
