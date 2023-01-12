FROM python:3.9-alpine@sha256:02311d686cd35b0f838854d6035c679acde2767a4fd09904e65355fbd9780f8a


EXPOSE 8000

# Create app directory
WORKDIR /usr/src/app

COPY . .
RUN mv fn/* .
RUN python -m pip install -r requirements.txt --user

CMD [ "python3", "functionhandler.py" ]
