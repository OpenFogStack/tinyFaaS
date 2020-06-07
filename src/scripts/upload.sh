#!/bin/bash

# upload.sh folder-name name threads


pushd $1 >/dev/null
curl http://localhost:8080/upload --data "{\"environment\": {\"ENVVAR\":\"value\" }, \"name\": \"$2\", \"threads\": $3, \"zip\": \"`zip -r - * | base64 -w 0`\"}"
popd >/dev/null