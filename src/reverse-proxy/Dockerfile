FROM golang:alpine as golang

WORKDIR /go/src/app
COPY . .

RUN apk update
RUN apk add --no-cache git
RUN go get -d -v ./...
RUN CGO_ENABLED=0 go install -v ./...

FROM scratch

EXPOSE 80/tcp
EXPOSE 5683/udp

COPY --from=golang /go/bin/app app

ENTRYPOINT [ "./app" ]
