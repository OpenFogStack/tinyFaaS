#!/bin/bash

set -e

for cont in $(docker ps -a -q --filter label=tinyfaas-function); do
    docker logs "$cont"
done
