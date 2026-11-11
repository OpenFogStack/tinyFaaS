ARG PYTHON_VERSION=3.13
ARG ALPINE_VERSION=3.24

FROM python:${PYTHON_VERSION}-alpine${ALPINE_VERSION}

COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

# Create app directory
WORKDIR /usr/src/app

COPY functionhandler.py .
