ARG GO_VERSION=1.27.3
ARG ALPINE_VERSION=3.24

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION}

WORKDIR /usr/src/app

COPY functionhandler.go ./
