FROM golang

EXPOSE 80/tcp
EXPOSE 5683/udp

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["app"]
