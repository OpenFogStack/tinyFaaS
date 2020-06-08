# tinyFaaS

If you are interested in this please check the original repo at https://github.com/OpenFogStack/tinyFaaS

## Setup Instructions

Please note that this will use your computers docker instance to manage containers and will allow anyone in your network to start docker-containers with arbitrary code. If you don't know what this means you do _not_ want to run this on your computer.

`cd` to the src directory and run `./scripts/start.sh`, after that a tinyFaaS instance will run on your host.  
To shut down tinyFaaS just run `./scripts/cleanup.sh`  
To get an overview of deployed functions run `./scripts/list.sh`  
To fetch logs run `./scripts/logs.sh`

## Changes of this fork

* uploading functions on runtime
* deleting functions on runtime
* listing functions
* aggregating logs
* accessing functions with http instead of coap
* methods other than GET supported
* functions can handle accesses to subpaths
* allowing to pass arguments/headers to functions (functions are handled with express.js)
* allowing to pass environment variables while deploying functions
* cleaning up after shutdown

## License

The code in this repository is licensed under the terms of the [MIT](./LICENSE) license.