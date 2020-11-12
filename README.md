# tinyFaaS

A Lightweight FaaS Platform for Edge Environments

If you use this software in a publication, please cite it as:

### Text

T. Pfandzelter and D. Bermbach, **tinyFaaS: A Lightweight FaaS Platform for Edge Environments**, 2020 IEEE International Conference on Fog Computing (ICFC), Sydney, Australia, 2020, pp. 17-24, doi: 10.1109/ICFC49376.2020.00011.

### BibTeX

```
@inproceedings{pfandzelter_tinyfaas:_2020,
	title = {tinyFaaS: A Lightweight FaaS Platform for Edge Environments},
	booktitle = {2020 IEEE International Conference on Fog Computing (ICFC)},
	author = {Pfandzelter, Tobias and Bermbach, David},
	year = {2020},
	publisher = {IEEE},
	pages = 17--24
}
```

For a full list of publications, please see [our website](https://www.mcc.tu-berlin.de/menue/forschung/publikationen/parameter/en/).

### License

The code in this repository is licensed under the terms of the [MIT](./LICENSE) license.

### Ports

* tcp 8080: management system, anyone who can access this can deploy arbitrary docker containers on your host system
* tcp 5683: http server, this is where your functions will be

## Instructions

Please note that this will use your computers docker instance to manage containers and will allow anyone in your network to start docker-containers with arbitrary code. If you don't know what this means you do _not_ want to run this on your computer.

To use tinyFaaS in the version used in the paper mentioned above, use `git checkout v0.1`.

To start this tinyFaaS implementation, simply build and start the management service in a Docker container.
It will then create the reverse-proxy in a separate container.

On constrained devices, you may run into issues when pulling the required Docker images for the first time.
Use these commands to pull these on a new installation (they will be cached for subsequent use):

```bash
docker pull python:3-alpine
docker pull golang:alpine
docker pull node:10-alpine
```

`cd` to the src directory and run `./scripts/start.sh`, after that a tinyFaaS instance will run on your host.  
To shut down tinyFaaS just run `./scripts/cleanup.sh`  
To get an overview of deployed functions run `./scripts/list.sh`  
To fetch logs run `./scripts/logs.sh`