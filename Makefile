PROJECT_NAME := "tinyFaaS"
PKG := "github.com/OpenFogStack/$(PROJECT_NAME)"
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v /ext/ | grep -v _test.go)
DOCKER0_GATEWAY := $(shell docker inspect -f '{{range .IPAM.Config}}{{.Gateway}}{{end}}' bridge)
CERTS_DIR := ./certs
TEST_DIR := ./test
ETCD_HOST := ${DOCKER0_GATEWAY}
ETCD_PORT := 2379
FRED_HOST := ${DOCKER0_GATEWAY}
FRED_PORT := 9001
FRED_PEERING_PORT := 5555

.PHONY: all build test start startkv etcd cleanetcd clean

all: build clean start

build: manager rproxy

manager rproxy: $(GO_FILES)
	@go build -o $@ -v $(PKG)/cmd/$@

start: manager rproxy
	./manager

startkv: fred manager rproxy ${CERTS_DIR}/ca.key ${CERTS_DIR}/ca.crt
	TF_BACKEND=dockerkv DOCKERKV_CERTS_DIR=${CERTS_DIR} DOCKERKV_CA_CERT_PATH=${CERTS_DIR}/ca.crt DOCKERKV_CA_KEY_PATH=${CERTS_DIR}/ca.key ./manager

fred: fred-compose.yml ${CERTS_DIR}/fred.key ${CERTS_DIR}/fred.crt ${CERTS_DIR}/etcd.key ${CERTS_DIR}/etcd.crt
	ETCD_HOST=${ETCD_HOST} ETCD_PORT=${ETCD_PORT} FRED_HOST=${FRED_HOST}  FRED_PORT=${FRED_PORT} FRED_PEERING_PORT=${FRED_PEERING_PORT} docker compose -f $< up -d

${CERTS_DIR}/ca.key:
	@mkdir -p ${CERTS_DIR}
	@openssl genrsa -out $@ 2048

${CERTS_DIR}/ca.crt: ${CERTS_DIR}/ca.key
	@mkdir -p ${CERTS_DIR}
	@openssl req -x509 -new -nodes -key $< -days 1825 -sha512 -out $@ -subj "/C=DE/L=Berlin/O=OpenFogStack/OU=tinyFaaS"

${CERTS_DIR}/etcd.key ${CERTS_DIR}/etcd.crt: ${CERTS_DIR}/ca.crt
	@mkdir -p ${CERTS_DIR}
	./gen-cert.sh ${CERTS_DIR} etcd ${ETCD_HOST}

${CERTS_DIR}/fred.key ${CERTS_DIR}/fred.crt: ${CERTS_DIR}/ca.crt
	@mkdir -p ${CERTS_DIR}
	./gen-cert.sh ${CERTS_DIR} fred ${FRED_HOST}

test: ${TEST_DIR}/test_all.py
	@python3 $<

testkv: ${TEST_DIR}/test_kv.py
	@python3 $<

clean: fred-compose.yml
	@docker rm -f $$(docker ps -a -q --filter label=tinyFaaS) > /dev/null || true
	@docker network rm $$(docker network ls -q --filter label=tinyFaaS) > /dev/null || true
	@docker rmi $$(docker image ls -q --filter label=tinyFaaS) > /dev/null || true
	@docker compose -f fred-compose.yml down
	@rm -rf ./tmp
