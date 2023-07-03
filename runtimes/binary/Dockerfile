FROM golang:1.20-alpine AS builder

WORKDIR /usr/src/build
COPY functionhandler.go .
RUN GO111MODULE=off CGO_ENABLED=0 go build -o handler.bin .

FROM alpine

EXPOSE 8000

# Create app directory
WORKDIR /usr/src/app

COPY --from=builder /usr/src/build/handler.bin .

COPY . .
RUN rm functionhandler.go
RUN mv fn/* .
RUN chmod +x fn.sh

CMD [ "./handler.bin" ]
