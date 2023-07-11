PROJECT_NAME := "tinyFaaS"
PKG := "github.com/OpenFogStack/$(PROJECT_NAME)"
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v /ext/ | grep -v _test.go)
TEST_DIR := ./test

.PHONY: all build start clean

all: build clean start

build: manager rproxy

manager rproxy: $(GO_FILES)
	@go build -o $@ -v $(PKG)/cmd/$@

start: manager rproxy
	./manager

test: build ${TEST_DIR}/test_all.py
	@python3 ${TEST_DIR}/test_all.py

clean:
	@docker rm -f $$(docker ps -a -q --filter label=tinyFaaS) > /dev/null || true
	@docker network rm $$(docker network ls -q --filter label=tinyFaaS) > /dev/null || true
	@docker rmi $$(docker image ls -q --filter label=tinyFaaS) > /dev/null || true
	@rm -rf ./tmp
