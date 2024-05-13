PROJECT_NAME := "tinyFaaS"
PKG := "github.com/OpenFogStack/$(PROJECT_NAME)"
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v /ext/ | grep -v _test.go)
TEST_DIR := ./test

OS=$(shell go env GOOS)
ARCH=$(shell go env GOARCH)

.PHONY: all build start clean

all: build

build: tinyfaas-${OS}-${ARCH}

# requires protoc,  protoc-gen-go and protoc-gen-go-grpc
# install from your package manager, e.g.:
# 	brew install protobuf
# 	brew install protoc-gen-go
#	brew install protoc-gen-go-grpc
pkg/grpc/tinyfaas/tinyfaas.pb.go pkg/grpc/tinyfaas/tinyfaas_grpc.pb.go: pkg/grpc/tinyfaas/tinyfaas.proto
	@protoc -I . $< --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=require_unimplemented_servers=false,paths=source_relative

# requires grpcio-tools and mypy-protobuf
# 	python3 -m pip install -r requirements.txt
pkg/grpc/tinyfaas/tinyfaas_pb2.py pkg/grpc/tinyfaas/tinyfaas_pb2.pyi pkg/grpc/tinyfaas/tinyfaas_pb2_grpc.py: pkg/grpc/tinyfaas/tinyfaas.proto
	@python3 -m grpc_tools.protoc -I . --python_out=. --grpc_python_out=. --mypy_out=. $<

cmd/manager/rproxy-%.bin: pkg/grpc/tinyfaas/tinyfaas_pb2.py pkg/grpc/tinyfaas/tinyfaas_pb2.pyi pkg/grpc/tinyfaas/tinyfaas_pb2_grpc.py pkg/grpc/tinyfaas/tinyfaas.pb.go pkg/grpc/tinyfaas/tinyfaas_grpc.pb.go $(GO_FILES)
	GOOS=$(word 1,$(subst -, ,$*)) GOARCH=$(word 2,$(subst -, ,$*)) go build -o $@ -v $(PKG)/cmd/rproxy

tinyfaas-%: cmd/manager/rproxy-%.bin $(GO_FILES)
	GOOS=$(word 1,$(subst -, ,$*)) GOARCH=$(word 2,$(subst -, ,$*)) go build -o $@ -v $(PKG)/cmd/manager

start: tinyfaas-${OS}-${ARCH}
	./$<

test: build ${TEST_DIR}/test_all.py
	@python3 ${TEST_DIR}/test_all.py

clean: clean.sh
	@sh clean.sh
