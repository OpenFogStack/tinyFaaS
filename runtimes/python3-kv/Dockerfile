FROM python:3.11-slim-buster

EXPOSE 8000

# Create app directory
WORKDIR /usr/src/app

COPY __kv-requirements.txt __kv-requirements.txt
RUN python -m pip install -r __kv-requirements.txt --user

COPY . .
RUN mv fn/* .
RUN python -m pip install -r requirements.txt --user

CMD [ "python3", "functionhandler.py" ]
