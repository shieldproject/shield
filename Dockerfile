FROM golang:1.11 as build

RUN apt-get update \
 && apt-get install -y bzip2 unzip curl openssh-client

RUN curl -sLo /bin/jq https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64 \
 && chmod 0755 /bin/jq

ARG VERSION
COPY / /go/src/github.com/shieldproject/shield/
RUN cd /go/src/github.com/shieldproject/shield \
 && make build BUILD_TYPE="build -ldflags='-X main.Version=$VERSION'"
RUN mkdir -p /dist/bin /dist/plugins \
 && mv /go/src/github.com/shieldproject/shield/shieldd \
       /go/src/github.com/shieldproject/shield/shield-agent \
       /go/src/github.com/shieldproject/shield/shield-crypt \
       /go/src/github.com/shieldproject/shield/shield-report \
       /go/src/github.com/shieldproject/shield/shield-schema \
       /go/src/github.com/shieldproject/shield/bin/shield-pipe \
       /dist/bin \
 && for plugin in $(cat /go/src/github.com/shieldproject/shield/plugins); do \
      cp /go/src/github.com/shieldproject/shield/$plugin /dist/plugins; \
    done \
 && cp -R /go/src/github.com/shieldproject/shield/web/htdocs /dist/ui

ADD init /dist/init
RUN chmod 0755 /dist/init/*

FROM ubuntu:19.04
RUN apt-get update \
 && apt-get install -y bzip2 curl openssh-client \
 && rm -rf /var/lib/apt/lists/*
COPY --from=build /dist /shield
