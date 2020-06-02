#!/bin/bash
./scripts/cleanup.sh
docker build -t tinyfaas-mgmt .
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 --name tinyfaas-mgmt -d tinyfaas-mgmt tinyfaas-mgmt

sleep 5

curl http://localhost:8080 --data '{"path": "sieve-of-erasthostenes", "resource": "/sieve/primes", "entry": "sieve.js", "threads": 4}'
#aiocoap-client coap://localhost/sieve/primes