#!/bin/sh

TF_TAG="tinyFaaS"
TMP_DIR="tmp"

# remove old containers, networks and images
containers=$(docker ps -a -q --filter label=$TF_TAG)

if [ -n "$containers" ]; then
    for container in $containers; do
        docker stop "$container" > /dev/null || echo "Failed to stop container $container! Please stop it manually..."
        docker rm "$container" > /dev/null || echo "Failed to remove container $container! Please remove it manually..."
    done
else
    echo "No old containers to remove. Skipping..."
fi

networks=$(docker network ls -q --filter label=$TF_TAG)

if [ -n "$networks" ]; then
    for network in $networks; do
        docker network rm "$network" > /dev/null || echo "Failed to remove network $network! Please remove it manually..."
    done
else
    echo "No old networks to remove. Skipping..."
fi

images=$(docker image ls -q --filter label=$TF_TAG)

if [ -n "$images" ]; then
    for image in $images; do
        docker image rm "$image" > /dev/null || echo "Failed to remove image $image! Please remove it manually..."
    done
else
    echo "No old images to remove. Skipping..."
fi

# remove tmp directory
if [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR" > /dev/null || echo "Failed to remove directory $TMP_DIR ! Please remove it manually..."
else
    echo "No tmp directory to remove. Skipping..."
fi
