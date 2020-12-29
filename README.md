# tinyFaaS: A Lightweight FaaS Platform for Edge Environments

tinyFaaS is a lightweight FaaS platform for edge environment with a focus on performance in constrained environments.

## Research

To use tinyFaaS in the version used in the paper mentioned above, use `git checkout v0.1`.
If you use this software in a publication, please cite it as:

### Text

T. Pfandzelter and D. Bermbach, **tinyFaaS: A Lightweight FaaS Platform for Edge Environments**, 2020 IEEE International Conference on Fog Computing (ICFC), Sydney, Australia, 2020, pp. 17-24, doi: 10.1109/ICFC49376.2020.00011.

### BibTeX

```bibtex
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

## Instructions

**Disclaimer**: Please note that this will use your computer's Docker instance to manage containers and will allow anyone in your network to start Docker containers with arbitrary code. If you don't know what this means you do _not_ want to run this on your computer.
Additionally, note that this software is provided as a research prototype and is not production ready.

### About

tinyFaaS comprises the _Management Service_, the _Reverse Proxy_, and a number of _Function Handlers_.
In order to run tinyFaaS, the Management Service has to be deployed.
It will then automatically start the Reverse Proxy.
Once a function is deployed to tinyFaaS, Function Handlers are created automatically.

### Getting Started

Before you get started, make sure you have Docker installed on the machine you want to run tinyFaaS on.

To get started, simply build and start the Management Service:

```bash
make
```

OR

```bash
docker build -t tinyfaas-mgmt ./src/
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 --name tinyfaas-mgmt -d tinyfaas-mgmt tinyfaas-mgmt
```

The Reverse Proxy will be built and started automatically.
Please note that you cannot use tinyFaaS until the Reverse Proxy is running.

### Managing Functions

To manage functions on tinyFaaS, use the included scripts included in `./src/scrips`.

To upload a function, run `upload.sh {FOLDER} {NAME} {THREADS}`, where `{FOLDER}` is the path to your function code, `{NAME}` is the name for your function, and `{THREADS}` is a number specifying the number of Function Handlers for your function.

To get a list of existing functions, run `list.sh`.

To delete a function, run `delete.sh {NAME}`, where `{NAME}` is the name of the function you want to remove.

Additionally, we provide scripts to read logs from your function and to wipe all functions from tinyFaaS.

### Writing Functions

This tinyFaaS prototype only supports functions written for NodeJS 10.
Your function must be supplied as a Node module with the name `fn` that exports a single function that takes the `req` and `res` parameters for request and response, respectively.
`res` supports the `send()` function that has one parameter, a string that is passed to the client as-is.

To get started with functions, use the example "Sieve of Erasthostenes" function in `./examples/sieve-of-erasthostenes`.

### Calling Functions

tinyFaaS supports different application layer protocols at its Reverse Proxy.
Different protocols are useful for different use-cases: CoAP for lightweight communication, e.g., for IoT devices; HTTP to support traditional web applications; GRPC for inter-process communication.

#### CoAP

To call a tinyFaaS function using its CoAP endpoint, make a GET or POST request to`coap://{HOST}:{PORT}/{NAME}` where `{HOST}` is the address of the tinyFaaS host, `{PORT}` is the port for the tinyFaaS CoAP endpoint (default is `5683`), and `{NAME}` is the name of your function. You may include data in any form you want, it will be passed to your function.

Unfortunately, [`curl` does not yet support CoAP](https://curl.se/mail/lib-2018-05/0017.html), but [a number of other tools are available](https://coap.technology/).

#### HTTP

To call a tinyFaaS function using its HTTP endpoint, make a GET or POST request to `http://{HOST}:{PORT}/{NAME}` where `{HOST}` is the address of the tinyFaaS host, `{PORT}` is the port for the tinyFaaS HTTP endpoint (default is `80`), and `{NAME}` is the name of your function. You may include data in any form you want, it will be passed to your function.

TLS is not supported.

#### GRPC

To use the GRPC endpoint, compile the `api` protocol buffer as included in `./src/reverse-proxy/api` and import it into your application.
Specify the tinyFaaS host and port (default is `8000`) for the GRPC endpoint and use the `Request` function with the `functionIdentifier` being your function's name and the `data` field including data in any form you want.

### Removing tinyFaaS

To remove tinyFaaS, remove all containers created by tinyFaaS and the Management Service, as well as the internal networks used by tinyFaaS:

```bash
make clean
```

OR

```bash
docker rm -f tinyfaas-mgmt 2>/dev/null
docker rm -f $(docker ps -a -q  --filter network=endpoint-net) 2>/dev/null
for line in $(docker network ls --filter name=handler-net -q) ; do
    docker rm -f $(docker ps -a -q  --filter network=$line)
done

docker network rm $(docker network ls -q --filter name=endpoint-net) 2>/dev/null
docker network rm $(docker network ls -q --filter name=handler-net) 2>/dev/null
```

### Specifying Ports

By default, tinyFaaS will use the following ports:

| Port | Protocol | Description        |
| ---- | -------- | ------------------ |
| 8080 | TCP      | Management Service |
| 5683 | UDP      | CoAP Endpoint      |
| 80   | TCP      | HTTP Endpoint      |
| 8000 | TCP      | GRPC Endpoint      |

To change the port of the management service, change the port binding in the `docker run` command.

To change or deactivate the endpoints of tinyFaaS, you can use the `COAP_PORT`, `HTTP_PORT`, and `GRPC_PORT` environment variables, which must be passed to the Management Service Docker container.
Specify `-1` to deactivate a specific endpoint.
For example, to use `6000` as the port for the CoAP and deactive GRPC, run the Management Service with this command:

```bash
docker run --env COAP_PORT=6000 --env GRPC_PORT=-1 -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 --name tinyfaas-mgmt -d tinyfaas-mgmt tinyfaas-mgmt
```
