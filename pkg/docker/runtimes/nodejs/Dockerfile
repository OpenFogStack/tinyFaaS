FROM scratch

ADD blob.tar.gz /

EXPOSE 8000

WORKDIR /usr/src/app

COPY fn fn

RUN npm install ./fn

CMD [ "node", "functionhandler.js" ]
