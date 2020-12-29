FROM python:3-alpine

WORKDIR /usr/src/app
EXPOSE 8080

RUN pip install --no-cache-dir tornado
RUN pip install --no-cache-dir docker

COPY . .

ENTRYPOINT [ "python", "./management-service.py" ]
CMD []
