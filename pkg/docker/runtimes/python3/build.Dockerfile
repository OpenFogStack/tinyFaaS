FROM python:3.11-alpine

# Create app directory
WORKDIR /usr/src/app

COPY functionhandler.py .
