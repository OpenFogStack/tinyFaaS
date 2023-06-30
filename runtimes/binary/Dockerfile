FROM python:3.11-alpine


EXPOSE 8000

# Create app directory
WORKDIR /usr/src/app

COPY . .
RUN mv fn/* .
RUN chmod +x fn.sh

CMD [ "python3", "functionhandler.py" ]
