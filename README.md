# tinyFaaS: A Lightweight FaaS Platform for Edge Environments

tinyFaaS is a lightweight FaaS (Function-as-a-Service) platform for edge
environment with a focus on performance in constrained environments.

## Research

To use tinyFaaS in the version used in our paper, use `git checkout v0.1`.
If you use this software in a publication, please cite it as:

### Text

T. Pfandzelter and D. Bermbach, **tinyFaaS: A Lightweight FaaS Platform for Edge
Environments**, Proceedings of the 2020 IEEE International Conference on Fog
Computing (ICFC '20), Sydney, Australia, 2020, pp. 17-24, DOI:
10.1109/ICFC49376.2020.00011.

### BibTeX

```bibtex
@inproceedings{pfandzelter_tinyfaas:_2020,
    author = "Pfandzelter, Tobias and Bermbach, David",
    title = "tinyFaaS: A Lightweight FaaS Platform for Edge Environments",
    booktitle = "Proceedings of the 2020 IEEE International Conference on Fog Computing (ICFC '20)",
    year = 2020,
    publisher = "IEEE",
    pages = "17--24",
    doi = "10.1109/ICFC49376.2020.00011"
}
```

For a full list of publications, please see [our
website](https://www.tu.berlin/en/3s/research/publications).

### License

The code in this repository is licensed under the terms of the [MIT](./LICENSE)
license.

## Instructions

> [!CAUTION]
> tinyFaaS uses the local Docker instance to manage containers, and it allows
> anyone on your network to start Docker containers running arbitrary code. If
> you don't understand what this means, you should not run this software on your
> computer. Also note that this is a research prototype and is not
> production-ready.

### About

tinyFaaS comprises the _management service_, the _reverse proxy_, and a number
of _function handlers_. In order to run tinyFaaS, the management service has to
be deployed. It will then automatically start the reverse proxy. Once a function
is deployed to tinyFaaS, function handlers are created automatically.

### Prerequisites

Before you get started, make sure you have the following dependencies installed:

- Go, to compile management service and reverse proxy,
- Docker (>=v29),
- Make,
- a writable directory (tinyFaaS writes temporary files to a `./tmp` directory).

You can see the exact required Go version in the [go.mod](./go.mod) file.

Note that tinyFaaS is intended for Linux hosts (`x86_64` and `arm64`). Due to
limitations of Docker Desktop for Mac, installing and running
[`docker-mac-net-connect`](https://github.com/chipmk/docker-mac-net-connect) is
necessary to run tinyFaaS on macOS hosts. Running tinyFaaS on Windows computers
(native or through WSL) is probably possible but has not been tested and is thus
not recommended.

### Getting Started

Start tinyFaaS with:

```sh
make start
```

This builds tinyFaaS for your host platform if necessary and runs it with
`sudo`, so you may be asked for your password.

The reverse proxy will be started automatically. Please note that you cannot use
tinyFaaS until the reverse proxy is running.

### Managing Functions

To manage functions on tinyFaaS, use the scripts included in `./scripts`.

#### Deploy a function

There are two ways to deploy a function:

```sh
$ ./scripts/upload.sh {FOLDER} {NAME} {ENV} {THREADS}
$ ./scripts/uploadURL.sh {URL} {SUBFOLDER_PATH} {NAME} {ENV} {THREADS}
```

The main difference is that `upload.sh` takes a local folder that contains your
function code, whereas `uploadURL.sh` deploys a function from a zip file that
the tinyFaaS manager can reach at the given URL. In that case, you also need to
give the folder that contains the code inside the zip file (use `/` if the code
is at the top level). `upload.sh` requires the `zip`, `base64`, and `curl`
utilities.

The other available options are:

* `{NAME}`: the name of the function, which also identifies it. You need this
  name to call the function.
* `{ENV}`: the runtime you want to use (`python3`, `go`, `nodejs`, `binary`).
* `{THREADS}`: the number of function handlers, which is how many identical
  containers are created for the function. Each container can already handle
  requests in parallel on its own, and the reverse proxy forwards a call to a
  random function handler. If you want to change this value, you need to
  re-deploy the function.

Examples:

```sh
$ ./scripts/upload.sh "./test/fns/sieve-of-eratosthenes" "sieve" "nodejs" 1
$ ./scripts/uploadURL.sh \
    "https://github.com/OpenFogStack/tinyFaaS/archive/main.zip" \
    "tinyFaaS-main/test/fns/sieve-of-eratosthenes" \
    "sieve" "nodejs" 1
```

#### Other commands

To get a list of existing functions, run `./scripts/list.sh`.

To delete a function, run `./scripts/delete.sh {NAME}`. To remove all
functions, run `./scripts/wipe-functions.sh`.

To read the logs of all functions, run `./scripts/logs.sh` or
`./scripts/get_logs.sh`. You can also ead the logs of a single function directly
with `curl "localhost:8080/logs?name={NAME}"`.

### Writing Functions

This tinyFaaS prototype supports functions written for Go 1.27, Node.js 26,
Python 3.13, and binary functions. A good place to start is the collection of
test functions in [`./test/fns`](./test/fns/). In Go, Node.js and Python
functions, HTTP headers and gRPC metadata are available as key-value pairs. See
the "show-headers" test functions for more information.

#### Go

Your function must be supplied as a file named `fn.go` that exposes the package
`main` with the function `fn` that is invoked for every function invocation.
This method must accept two arguments: `data` of type `string`, and `headers` of
type `map[string]string`. Must return a string and an error as return values.
You may also provide a `go.mod` file from which dependencies will be installed
alongside your function.

#### Node.js

Your function must be supplied as a Node module with the name `fn` that exports
a single function that takes the `req` and `res` parameters for request and
response, respectively. `res` supports the `send()` function that has one
parameter, a string that is passed to the client as-is.

#### Python

Your function must be supplied as a file named `fn.py` that exposes a method
`fn` that is invoked for every function invocation. This method must accept two
arguments: the input as a string (that can also be `None`) and the headers as a
dictionary. It must return a string or `None`. You may also provide a
`requirements.txt` file from which dependencies will be installed alongside your
function. Any other data you provide will be available.

#### Binary

Your function must be provided as a `fn.sh` shell script that is invoked for
every function call. This shell script may also call other binaries as needed.
Input data is provided from `stdin`. Output responses should be provided on
`stdout`. Anything written to `stderr` is also included in the response.

### Calling Functions

tinyFaaS supports different application layer protocols at its reverse proxy.
Different protocols are useful for different use-cases: CoAP for lightweight
communication, e.g., for IoT devices; HTTP to support traditional web
applications; GRPC for inter-process communication.

#### CoAP

To call a tinyFaaS function using its CoAP endpoint, make a GET or POST request
to `coap://{HOST}:{PORT}/{NAME}` where `{HOST}` is the address of the tinyFaaS
host, `{PORT}` is the port for the tinyFaaS CoAP endpoint (default is `5683`),
and `{NAME}` is the name of your function. You may include data in any form you
want, it will be passed to your function.

Unfortunately, [`curl` does not yet support
CoAP](https://curl.se/mail/lib-2018-05/0017.html), but [a
number](https://github.com/coapjs/coap-cli) [of
other](https://aiocoap.readthedocs.io/en/latest/tools.html) [tools are
available](https://fitbit.github.io/golden-gate/tools/coap_client.html).

#### HTTP

To call a tinyFaaS function using its HTTP endpoint, make a GET or POST request
to `http://{HOST}:{PORT}/{NAME}` where `{HOST}` is the address of the tinyFaaS
host, `{PORT}` is the port for the tinyFaaS HTTP endpoint (default is `8000`),
and `{NAME}` is the name of your function. You may include data in any form you
want, it will be passed to your function.

TLS is not supported (but contributions are welcome).

To make an asynchronous request, pass the `X-tinyFaaS-Async` header with any
value. An asynchronous request means the client will receive a `202` response
code immediately and no function results will be sent back.

```bash
curl --header "X-tinyFaaS-Async: true" "http://localhost:8000/sieve"
```

#### gRPC

To use the gRPC endpoint, compile the `tinyfaas` protocol buffer (included in
[`./pkg/grpc/tinyfaas`](./pkg/grpc/tinyfaas)) for your programming language and
import it into your application. We already provide compiled versions for Go and
Python in that directory. Specify the tinyFaaS host and port (default is `9000`)
for the GRPC endpoint and use the `Request` function with the
`functionIdentifier` being your function's name and the `data` field including
data in any form you want.

### Removing tinyFaaS

When you stop the management service with `SIGINT` (`Ctrl+C`), the reverse proxy
and all function handlers should be stopped. You can also use:

```sh
make clean
```

OR

```sh
docker rm -f $(docker ps -a -q --filter label=tinyFaaS)
docker network rm $(docker network ls -q --filter label=tinyFaaS)
docker rmi $(docker image ls -q --filter label=tinyFaaS)
rm -rf ./tmp
```

You can also remove the files generated to build tinyFaaS with:

```sh
make clean-generated
```

### Specifying Ports

By default, tinyFaaS will use the following ports:

| Port | Protocol | Description        |
| ---- | -------- | ------------------ |
| 8080 | TCP      | Management Service |
| 5683 | UDP      | CoAP Endpoint      |
| 8000 | TCP      | HTTP Endpoint      |
| 9000 | TCP      | GRPC Endpoint      |

The scripts in `./scripts` expect the management service on port `8080`.

To change or deactivate the endpoints of tinyFaaS, you can use the `coap_PORT`,
`http_PORT`, and `grpc_PORT` environment variables (the names are
case-sensitive), which must be passed to the management service. Specify `-1` to
deactivate a specific endpoint. `sudo` does not pass your environment variables
on by default, so set them in the command that runs the binary rather than
using `make start`. For example, to use `6000` as the port for CoAP and
deactivate GRPC, run:

```sh
make build
sudo env coap_PORT=6000 grpc_PORT=-1 ./tinyfaas-$(go env GOOS)-$(go env GOARCH)
```

### Tests

The tests in [`./test`](./test) test the end-to-end functionality of tinyFaaS
and are expected to complete successfully. We use these tests during development
to ensure no patches break any functionality. The tests can also serve as
documentation on the expected behavior of tinyFaaS.

Running the tests requires:

- Python >3.10 with the `venv` module
- a tinyFaaS binary built for your host

Create a virtual environment for Python and install the necessary dependencies
for CoAP and gRPC:

```sh
python3 -m venv .venv
source .venv/bin/activate
python3 -m pip install -r test/requirements.txt
```

If you do not install these requirements, test runs are limited to invocations
with HTTP.

Run the tests with:

```sh
$ make test
.............
----------------------------------------------------------------------
Ran 13 tests in 28.063s

OK
```

Tests will output a `.` (dot) for a successful test, `F` for a failed assertion,
`E` for an unexpected error, `s` for a skipped test, `x` for an expected failure
and `u` for an unexpected success. tinyFaaS output will be written to
`tf_test.out`.

To see the details of each test, run make like this:

```sh
$ make test VERBOSE_TESTS=true
```

On macOS,
[`docker-mac-net-connect`](https://github.com/chipmk/docker-mac-net-connect) is
necessary to run tinyFaaS. There is a [known issue in
`docker-mac-net-connect`](https://github.com/chipmk/docker-mac-net-connect/issues/36)
that silently breaks the tunnel when Docker Desktop enters its resource saver
mode. If you find that tinyFaaS does not start properly on your macOS host, try
restarting the tunnel:

```sh
sudo brew services restart docker-mac-net-connect
```

### Known limitations

#### HTTP status code and headers

Any function, in any runtime, cannot control the HTTP status code or the HTTP
headers that are returned. Only the body is sent back to the client. If the
function returns an error or throws an exception, the HTTP status is still 200
and the body contains the error instead of the function's result. In the binary
runtime, the body contains only the exit error (for example, `exit status 1`),
not the output of the script.

To recap, for function requests, tinyFaaS returns the following HTTP status
codes to the client:

* `200 OK`: The container responded with any status: success, a Python
  exception, a Go error, a non-zero exit from a binary, or a Node.js
  `res.status(...)`.
* `202 Accepted`: The request had the `X-tinyFaaS-Async` header. The proxy
  replies straight away and never waits for the function.
* `404 Not Found`: No function is deployed under that name.
* `500 Internal Server Error`: The proxy couldn't read the client's request
  body, couldn't reach the container (connection refused, or the container
  crashed mid-request), or couldn't read the container's response.

#### Function timeout

The reverse proxy does not set a timeout for a function request. A synchronous
function that hangs keeps the client waiting forever instead of getting an error
code.
