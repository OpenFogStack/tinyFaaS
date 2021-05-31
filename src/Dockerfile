FROM python:3.9-alpine@sha256:02311d686cd35b0f838854d6035c679acde2767a4fd09904e65355fbd9780f8a

WORKDIR /usr/src/app
EXPOSE 8080

COPY . .

RUN python -m pip install -r requirements.txt --user

ENTRYPOINT [ "python", "./management-service.py" ]
CMD []
