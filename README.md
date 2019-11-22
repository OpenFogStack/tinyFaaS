# tinyFaaS
A Lightweight FaaS Platform for Edge Environments

## Build

To start this tinyFaaS implementation, simply build and start the management service in a Docker container. It will then create the gateway in a separate container.

To build the management service container, run:
`docker build -t tinyfaas-mgmt .`

Then start the container with:
`docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 --name tinyfaas-mgmt -d tinyfaas-mgmt tinyfaas-mgmt`

This ensures that the management service has access to Docker on the host and it will then expose port 8080 to accept incoming request.

To deploy a function (e.g. the "Sieve of Erasthostenes"), run:
`curl http://localhost:8080 --data '{"path": "sieve-of-erasthostenes", "resource": "/sieve/primes", "entry": "sieve.js", "threads": 4}' -v`
