#!/bin/bash

#delete.sh function-name

curl http://localhost:8080/delete --data "$1"
