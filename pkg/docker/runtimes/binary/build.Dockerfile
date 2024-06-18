ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.19

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

WORKDIR /usr/src/build
COPY functionhandler.go .
RUN GO111MODULE=off CGO_ENABLED=0 go build -o handler.bin .

FROM alpine:${ALPINE_VERSION}

# Create app directory
WORKDIR /usr/src/app

COPY --from=builder /usr/src/build/handler.bin .
