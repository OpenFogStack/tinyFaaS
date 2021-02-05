#!/bin/bash

# uploadURL.sh url subfolder name threads

curl http://localhost:8080/uploadURL --data "{\"name\": \"$3\", \"threads\": $4,\"url\": \"$1\",\"subfolder_path\": \"$2\"}"
