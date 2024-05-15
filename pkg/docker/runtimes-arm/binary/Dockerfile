FROM scratch

ADD blob.tar.gz /

EXPOSE 8000

WORKDIR /usr/src/app

COPY fn/* ./
RUN chmod +x fn.sh

CMD [ "./handler.bin" ]
