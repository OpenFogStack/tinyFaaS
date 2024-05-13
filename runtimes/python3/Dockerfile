FROM python:3.11-alpine


EXPOSE 8000

# Create app directory
WORKDIR /usr/src/app

COPY . .
RUN mv fn/* .
RUN python -m pip install -r requirements.txt --user

ENV PYTHONUNBUFFERED=1
CMD [ "python3", "functionhandler.py" ]
