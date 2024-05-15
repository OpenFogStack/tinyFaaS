FROM golang:1.22-alpine AS builder

WORKDIR /usr/src/build
COPY functionhandler.go .
RUN GO111MODULE=off CGO_ENABLED=0 go build -o handler.bin .

FROM alpine:3.19

# Create app directory
WORKDIR /usr/src/app

COPY --from=builder /usr/src/build/handler.bin .
