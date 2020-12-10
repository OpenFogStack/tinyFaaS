#!/bin/bash
./scripts/cleanup.sh

docker pull python:3-alpine
docker pull golang:alpine
docker pull node:10-alpine

docker build -t tinyfaas-mgmt .
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 --name tinyfaas-mgmt -d tinyfaas-mgmt tinyfaas-mgmt

sleep 5

#./scripts/upload.sh ../examples/sieve-of-erasthostenes/ /sieve/primes 4
#sleep 1

#curl localhost:5683/sieve/primes
