#!/bin/bash

#delete.sh function-name

set -e

if ! command -v curl &> /dev/null
then
    echo "curl could not be found but is a pre-requisite for this script"
    exit
fi

curl http://localhost:8080/delete --data "{\"name\": \"$1\"}"
