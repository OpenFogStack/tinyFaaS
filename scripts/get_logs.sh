#!/bin/bash

set -e

for line in $(docker network ls --filter name=handler-net -q) ; do
    for cont in $(docker ps -a -q  --filter network="$line") ; do
        docker logs "$cont"
    done
done
