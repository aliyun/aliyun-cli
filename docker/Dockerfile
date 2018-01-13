FROM python:2-alpine3.7

LABEL maintainer "Maik Ellerbrock <opensource@frapsoft.com>"

ENV VERSION 0.0.1

ARG SERVICE_USER

ENV SERVICE_USER ${SERVICE_USER:-aliyun}

COPY requirements.txt /tmp

RUN \
  adduser -s /sbin/nologin -u 1000 -H -D ${SERVICE_USER} && \
  apk add --no-cache build-base && \
  pip install aliyuncli && \
  pip install --no-cache-dir -r /tmp/requirements.txt && \
  apk del build-base && \
  rm -rf /tmp/*

USER ${SERVICE_USER}

WORKDIR /usr/local/bin

ENTRYPOINT [ "aliyuncli" ]

CMD        [ "--help" ]

