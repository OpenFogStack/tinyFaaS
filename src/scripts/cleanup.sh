#!/bin/bash
docker rm -f $(docker ps -a -q  --filter network=endpoint-net) 2>/dev/null
for line in $(docker network ls --filter name=handler-net -q) ; do
    docker rm -f $(docker ps -a -q  --filter network=$line)
done

docker network rm $(docker network ls -q --filter name=endpoint-net) 2>/dev/null
docker network rm $(docker network ls -q --filter name=handler-net) 2>/dev/null

