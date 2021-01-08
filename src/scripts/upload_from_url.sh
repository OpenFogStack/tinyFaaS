#!/bin/bash

# upload.sh url name threads subfolder_path base64_encoded

curl http://localhost:8080/uploadFromUrl --data "{\"environment\": {\"ENVVAR\":\"value\" }, \"name\": \"$2\", \"threads\": $3, \"url\": \"$1\", \"subfolder_path\": \"$4\", \"base64_encoded\": $5}"
