FROM golang:1.22-alpine3.19

ADD blob.tar.gz /

WORKDIR /usr/src/app

COPY fn/* ./

RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 go build -o handler .

EXPOSE 8000

CMD ["./handler"]