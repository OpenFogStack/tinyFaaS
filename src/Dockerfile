FROM python:3

WORKDIR /usr/src/app

RUN pip install --no-cache-dir tornado
RUN pip install --no-cache-dir docker

COPY . .
EXPOSE 8080

ENTRYPOINT [ "python", "./management-service.py" ]
CMD []
