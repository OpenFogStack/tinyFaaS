#!/bin/sh

set -e

TF_TAG="tinyFaaS"
TMP_DIR="tmp"

# remove old containers, networks and images
containers=$(docker ps -a -q --filter label=$TF_TAG)

if [ -n "$containers" ]; then
    docker stop "$containers" > /dev/null
    docker rm "$containers" > /dev/null
else
    echo "No old containers to remove. Skipping..."
fi

networks=$(docker network ls -q --filter label=$TF_TAG)

if [ -n "$networks" ]; then
    docker network rm "$networks" > /dev/null
else
    echo "No old networks to remove. Skipping..."
fi

images=$(docker image ls -q --filter label=$TF_TAG)

if [ -n "$images" ]; then
    for image in $images; do
        docker image rm "$image" > /dev/null
    done
else
    echo "No old images to remove. Skipping..."
fi

# remove tmp directory
if [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
else
    echo "No tmp directory to remove. Skipping..."
fi

set +e
