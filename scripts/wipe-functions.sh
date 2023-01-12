#!/bin/bash

#wipe-functions.sh

set -e

if ! command -v curl &> /dev/null
then
    echo "curl could not be found but is a pre-requisite for this script"
    exit
fi

curl http://localhost:8080/wipe --data ""
