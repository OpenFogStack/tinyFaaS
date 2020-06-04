# tinyFaaS

A Lightweight FaaS Platform for Edge Environments

If you use this software in a publication, please cite it as:

### Text
Tobias, David Bermbach. **tinyFaaS: A Lightweight FaaS Platform for Edge Environments**. In: Proceedings of the 2nd IEEE International Conference on Fog Computing 2020 (ICFC 2020). IEEE 2020.

### BibTeX
```
@inproceedings{pfandzelter_tinyfaas:_2020,
	title = {tinyFaaS: A Lightweight FaaS Platform for Edge Environments},
	booktitle = {Proceedings of the Second {IEEE} {International} {Conference} on {Fog} {Computing} 2020 (ICFC 2020)},
	author = {Pfandzelter, Tobias and Bermbach, David},
	year = {2020},
	publisher = {IEEE}
}
```

For a full list of publications, please see [our website](https://www.mcc.tu-berlin.de/menue/forschung/publikationen/parameter/en/).

## License

The code in this repository is licensed under the terms of the [MIT](./LICENSE) license.

## Instructions

To start this tinyFaaS implementation, simply build and start the management service in a Docker container. It will then create the gateway in a separate container.

To build the management service container, run:
`docker build -t tinyfaas-mgmt .`

Then start the container with:
`docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 --name tinyfaas-mgmt -d tinyfaas-mgmt tinyfaas-mgmt`

This ensures that the management service has access to Docker on the host and it will then expose port 8080 to accept incoming request.

To deploy a function (e.g. the "Sieve of Erasthostenes"), run:
`curl http://localhost:8080 --data '{"path": "sieve-of-erasthostenes", "resource": "/sieve/primes", "entry": "sieve.js", "threads": 4}' -v`
