FROM python:3.11-alpine

WORKDIR /usr/src/app
EXPOSE 8080

COPY . .

RUN python -m pip install -r requirements.txt --user

ENTRYPOINT [ "python", "./management-service.py" ]
CMD []
