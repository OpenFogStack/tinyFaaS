FROM golang:1.16.4-alpine@sha256:0dc62c5cc2d97657c17ff3bc0224214e10226e245c94317e352ee8a2c54368b4 as builder

WORKDIR /go/src/app
COPY . .

RUN apk update
RUN apk add --no-cache git
RUN go get -d -v ./...
RUN CGO_ENABLED=0 go install -v ./...

FROM scratch

EXPOSE 80/tcp
EXPOSE 6000/udp
EXPOSE 7000/tcp
EXPOSE 8000/tcp

COPY --from=builder /go/bin/reverse-proxy app

ENTRYPOINT [ "./app" ]

