#https://nodejs.org/en/docs/guides/nodejs-docker-webapp/
FROM node:20-alpine

EXPOSE 8000

# Create app directory
WORKDIR /usr/src/app
RUN npm cache add express

COPY . .
RUN npm install express
RUN npm install body-parser
RUN npm install ./fn

CMD [ "node", "functionhandler.js" ]
