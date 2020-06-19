FROM python:3-alpine

WORKDIR /usr/src/app

COPY . .

RUN pip install --no-cache-dir tornado
RUN pip install --no-cache-dir docker

EXPOSE 8080

ENTRYPOINT [ "python", "./management-service.py" ]
CMD []
