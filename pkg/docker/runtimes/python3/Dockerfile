FROM scratch

ADD blob.tar.gz /

EXPOSE 8000

# Create app directory
WORKDIR /usr/src/app

COPY fn/* ./

RUN uv pip install --system --no-cache --link-mode=copy -r requirements.txt

ENV PYTHONUNBUFFERED=1
CMD [ "python3", "functionhandler.py" ]
