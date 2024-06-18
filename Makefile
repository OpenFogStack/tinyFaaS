PROJECT_NAME := "tinyFaaS"
PKG := "github.com/OpenFogStack/$(PROJECT_NAME)"
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v /ext/ | grep -v _test.go)
TEST_DIR := ./test

SUPPORTED_ARCH=amd64 arm arm64
RUNTIMES := $(shell find pkg/docker/runtimes -name Dockerfile | xargs -n1 dirname | xargs -n1 basename)

OS=$(shell go env GOOS)
ARCH=$(shell go env GOARCH)

.PHONY: all
all: build

.PHONY: build
build: tinyfaas-${OS}-${ARCH}

.PHONY: start
start: tinyfaas-${OS}-${ARCH}
	./$<

.PHONY: test
test: build ${TEST_DIR}/test_all.py pkg/grpc/tinyfaas/tinyfaas_pb2.py pkg/grpc/tinyfaas/tinyfaas_pb2.pyi pkg/grpc/tinyfaas/tinyfaas_pb2_grpc.py
	@python3 ${TEST_DIR}/test_all.py

.PHONY: clean
clean: clean.sh
	@sh clean.sh


define arch_build
pkg/docker/runtimes-$(arch): $(foreach runtime,$(RUNTIMES),pkg/docker/runtimes-$(arch)/$(runtime))
endef
$(foreach arch,$(SUPPORTED_ARCH),$(eval $(arch_build)))

define runtime_build
.PHONY: pkg/docker/runtimes-$(arch)/$(runtime)
pkg/docker/runtimes-$(arch)/$(runtime): pkg/docker/runtimes-$(arch)/$(runtime)/Dockerfile pkg/docker/runtimes-$(arch)/$(runtime)/blob.tar.gz

pkg/docker/runtimes-$(arch)/$(runtime)/blob.tar.gz: pkg/docker/runtimes/$(runtime)/build.Dockerfile
	mkdir -p $$(@D)
	cd $$(<D) ; docker build --platform=linux/$(arch) -t tf-build-$(arch)-$(runtime) -f $$(<F) .
	docker run -d -t --platform=linux/$(arch) --name $${PROJECT_NAME}-$(runtime) --rm tf-build-$(arch)-$(runtime)
	docker export $${PROJECT_NAME}-$(runtime) | gzip > $$@
	docker kill $${PROJECT_NAME}-$(runtime)

pkg/docker/runtimes-$(arch)/$(runtime)/Dockerfile: pkg/docker/runtimes/$(runtime)/Dockerfile
	mkdir -p $$(@D)
	cp -r pkg/docker/runtimes/$(runtime)/Dockerfile $$@
endef
$(foreach arch,$(SUPPORTED_ARCH),$(foreach runtime,$(RUNTIMES),$(eval $(runtime_build))))

# requires protoc,  protoc-gen-go and protoc-gen-go-grpc
# install from your package manager, e.g.:
# 	brew install protobuf
# 	brew install protoc-gen-go
#	brew install protoc-gen-go-grpc
pkg/grpc/tinyfaas/tinyfaas.pb.go pkg/grpc/tinyfaas/tinyfaas_grpc.pb.go: pkg/grpc/tinyfaas/tinyfaas.proto
	@protoc -I $(<D) $< --go_out=$(<D) --go_opt=paths=source_relative --go-grpc_out=$(<D) --go-grpc_opt=require_unimplemented_servers=false,paths=source_relative

# requires grpcio-tools and mypy-protobuf
# 	python3 -m pip install -r requirements.txt
pkg/grpc/tinyfaas/tinyfaas_pb2.py pkg/grpc/tinyfaas/tinyfaas_pb2.pyi pkg/grpc/tinyfaas/tinyfaas_pb2_grpc.py: pkg/grpc/tinyfaas/tinyfaas.proto
	@python3 -m grpc_tools.protoc -I $(<D) --python_out=$(<D) --grpc_python_out=$(<D) --mypy_out=$(<D) $<

cmd/manager/rproxy-%.bin: pkg/grpc/tinyfaas/tinyfaas_pb2.py pkg/grpc/tinyfaas/tinyfaas_pb2.pyi pkg/grpc/tinyfaas/tinyfaas_pb2_grpc.py pkg/grpc/tinyfaas/tinyfaas.pb.go pkg/grpc/tinyfaas/tinyfaas_grpc.pb.go $(GO_FILES)
	GOOS=$(word 1,$(subst -, ,$*)) GOARCH=$(word 2,$(subst -, ,$*)) go build -o $@ -v $(PKG)/cmd/rproxy

tinyfaas-darwin-%: cmd/manager/rproxy-darwin-%.bin pkg/docker/runtimes-% $(GO_FILES)
	GOOS=darwin GOARCH=$* go build -o $@ -v $(PKG)/cmd/manager

tinyfaas-linux-%: cmd/manager/rproxy-linux-%.bin pkg/docker/runtimes-% $(GO_FILES)
	GOOS=linux GOARCH=$* go build -o $@ -v $(PKG)/cmd/manager
