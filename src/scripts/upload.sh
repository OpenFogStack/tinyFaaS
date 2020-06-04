#!/bin/bash

# upload.sh folder-name path threads



curl http://localhost:8080/upload --data "{\"resource\": \"$2\", \"threads\": $3, \"tarball\": \"`tar -cvf - -C $1 . | base64 -w 0`\"}"
