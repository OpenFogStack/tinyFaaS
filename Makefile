.PHONY: all start prepare clean

all: clean prepare start

prepare: ## Prepare tinyFaaS
	@docker build -t tinyfaas-mgmt ./src/
	@docker build -t tinyfaas-reverse-proxy ./src/reverse-proxy/
	@docker pull node:10-alpine@sha256:dc98dac24efd4254f75976c40bce46944697a110d06ce7fa47e7268470cf2e28

start: ## Start tinyFaaS
	@docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 --name tinyfaas-mgmt -d tinyfaas-mgmt tinyfaas-mgmt

clean: ## Remove tinyFaaS
	@docker rm -f tinyfaas-mgmt > /dev/null || true
	@docker rm -f $$(docker ps -a -q --filter network=endpoint-net) > /dev/null || true
	@docker rm -f $$(docker ps -a -q --filter network=handler-net) > /dev/null || true
	for line in $$(docker network ls -q --filter name=handler-net) ; do \
		docker rm -f $$(docker ps -a -q  --filter network=$$line) > /dev/null || true ; \
	done
	@docker network rm $$(docker network ls -q --filter name=endpoint-net) > /dev/null || true
	@docker network rm $$(docker network ls -q --filter name=handler-net) > /dev/null || true
