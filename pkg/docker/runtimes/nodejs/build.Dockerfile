#https://nodejs.org/en/docs/guides/nodejs-docker-webapp/
FROM node:20-alpine

# Create app directory
WORKDIR /usr/src/app

COPY functionhandler.js .
COPY package.json .

RUN npm install express && \
    npm install body-parser && \
    npm cache clean --force
