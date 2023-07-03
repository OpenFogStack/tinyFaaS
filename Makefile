PROJECT_NAME := "tinyFaaS"
PKG := "github.com/OpenFogStack/$(PROJECT_NAME)"
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v /ext/ | grep -v _test.go)

.PHONY: all build start clean

all: build clean start

build: manager rproxy

manager rproxy: $(GO_FILES)
	@go build -o $@ -v $(PKG)/cmd/$@

start:
	./manager

clean:
	@docker rm -f $$(docker ps -a -q --filter label=tinyFaaS) > /dev/null || true
	@docker network rm $$(docker network ls -q --filter label=tinyFaaS) > /dev/null || true
	@docker rmi $$(docker image ls -q --filter label=tinyFaaS) > /dev/null || true
	@rm -rf ./tmp
