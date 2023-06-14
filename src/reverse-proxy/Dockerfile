FROM golang:1.20.5-alpine as builder

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
