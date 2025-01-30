ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.19

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION}

WORKDIR /usr/src/app

COPY functionhandler.go ./