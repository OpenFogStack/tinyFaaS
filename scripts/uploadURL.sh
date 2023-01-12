#!/bin/bash

# uploadURL.sh url subfolder name env threads

set -e

if ! command -v curl &> /dev/null
then
    echo "curl could not be found but is a pre-requisite for this script"
    exit
fi

curl http://localhost:8080/uploadURL --data "{\"name\": \"$3\", \"env\": \"$4\",\"threads\": $5,\"url\": \"$1\",\"subfolder_path\": \"$2\"}"
