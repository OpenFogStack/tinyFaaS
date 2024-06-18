ARG PYTHON_VERSION=3.11
ARG ALPINE_VERSION=3.19

FROM python:${PYTHON_VERSION}-alpine${ALPINE_VERSION}

# Create app directory
WORKDIR /usr/src/app

COPY functionhandler.py .
