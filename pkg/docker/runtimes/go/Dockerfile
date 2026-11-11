ARG GO_VERSION=1.27.3
ARG ALPINE_VERSION=3.24

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION}

ADD blob.tar.gz /

WORKDIR /usr/src/app

COPY fn/* ./

RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 go build -o handler .

EXPOSE 8000

CMD ["./handler"]
