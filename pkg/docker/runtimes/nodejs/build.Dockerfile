ARG NODE_VERSION=26.12
ARG ALPINE_VERSION=3.24

FROM node:${NODE_VERSION}-alpine${ALPINE_VERSION}

# Create app directory
WORKDIR /usr/src/app

COPY functionhandler.js .
COPY package.json .

RUN npm install express && \
    npm install body-parser && \
    npm cache clean --force
