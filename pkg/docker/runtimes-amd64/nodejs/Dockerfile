FROM scratch

ADD blob.tar.gz /

EXPOSE 8000

WORKDIR /usr/src/app

COPY . .

RUN npm install ./fn

CMD [ "node", "functionhandler.js" ]
