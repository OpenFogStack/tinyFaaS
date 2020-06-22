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

## License

The code in this repository is licensed under the terms of the [MIT](./LICENSE) license.

## Instructions

To use tinyFaaS in the version used in the paper mentioned above, use `git checkout v0.1`.

To start this tinyFaaS implementation, simply build and start the management service in a Docker container.
It will then create the reverse-proxy in a separate container.

On constrained devices, you may run into issues when pulling the required Docker images for the first time.
Use these commands to pull these on a new installation (they will be cached for subsequent use):

```bash
docker pull python:3-alpine
docker pull golang:alpine
docker pull node:8-alpine
```

To build the management service container, run:

```bash
cd src
docker build -t tinyfaas-mgmt .
```

Then start the management service container with:

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 5000:8080 --name tinyfaas-mgmt -d tinyfaas-mgmt tinyfaas-mgmt
```

This ensures that the management service has access to Docker on the host and it will then expose port 5000 to accept incoming request.
When starting the management service, it will first build and deploy the reverse proxy as a second Docker container.
Depending on the performance of your host, this can take between a few seconds and up to a minute (on a Raspberry Pi 3B+).

To deploy a function (e.g. the "Sieve of Erasthostenes"), run:

```bash
curl http://localhost:5000 --data '{"path": "sieve-of-erasthostenes", "resource": "/sieve/primes", "entry": "sieve.js", "threads": 4}' -v
```

The reverse proxy will then expose this service on port 5683 (default CoAP port) as `coap://localhost:5683/sieve/primes`.
To change the default port, use the additional port parameter when running the tinyFaaS management service (e.g., to change the endpoint port to 7000):

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 --name tinyfaas-mgmt -d tinyfaas-mgmt tinyfaas-mgmt 7000
```

To stop and remove all containers on your system (including, **but not limited to**, containers managed by tinyFaaS), use:

```bash
docker stop $(docker ps -a -q)
docker rm $(docker ps -a -q)
```