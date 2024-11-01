ARG PYTHON_VERSION=3.11
# ARG ALPINE_VERSION=3.20

# FROM python:${PYTHON_VERSION}-alpine${ALPINE_VERSION}
FROM python:${PYTHON_VERSION}-slim

# Create app directory
WORKDIR /usr/src/app

# require cargo and rust
# RUN apk add --no-cache cargo rust --repository=https://dl-cdn.alpinelinux.org/alpine/edge/main
RUN python3 -m pip install robyn
# RUN apk del cargo rust

COPY functionhandler.py .
