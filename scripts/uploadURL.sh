#!/bin/bash

# uploadURL.sh url subfolder name threads

if ! command -v curl &> /dev/null
then
    echo "curl could not be found but is a pre-requisite for this script"
    exit
fi

curl http://localhost:8080/uploadURL --data "{\"name\": \"$3\", \"threads\": $4,\"url\": \"$1\",\"subfolder_path\": \"$2\"}"
