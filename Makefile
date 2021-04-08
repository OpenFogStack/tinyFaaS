.PHONY: start clean

start: ## Start tinyFaaS
	@docker build -t tinyfaas-mgmt ./src/
	@docker build ./src/reverse-proxy
	@docker pull node:10-alpine
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
