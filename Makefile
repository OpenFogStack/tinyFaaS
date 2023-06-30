.PHONY: all build start clean

all: build clean start

build: manager rproxy

manager: ./cmd/manager/main.go ./cmd/manager/util.go
	@go build -o ./manager ./cmd/manager/*.go

rproxy: ./cmd/rproxy/main.go
	@go build -o ./rproxy ./cmd/rproxy/*.go

start:
	./manager

clean:
	@docker rm -f $$(docker ps -a -q --filter label=tinyFaaS) > /dev/null || true
	@docker network rm $$(docker network ls -q --filter label=tinyFaaS) > /dev/null || true
	@docker rmi $$(docker image ls -q --filter label=tinyFaaS) > /dev/null || true
	@rm -rf ./tmp
