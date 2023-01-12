#https://nodejs.org/en/docs/guides/nodejs-docker-webapp/
FROM node:10-alpine@sha256:dc98dac24efd4254f75976c40bce46944697a110d06ce7fa47e7268470cf2e28

EXPOSE 8000

# Create app directory
WORKDIR /usr/src/app
RUN npm cache add express

COPY . .
RUN npm install express
RUN npm install ./fn

CMD [ "node", "functionhandler.js" ]
