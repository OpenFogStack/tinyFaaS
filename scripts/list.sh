#!/bin/bash

set -e

if ! command -v curl &> /dev/null
then
    echo "curl could not be found but is a pre-requisite for this script"
    exit
fi

curl localhost:8080/list
