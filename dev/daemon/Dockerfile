FROM postgres:9.6
ENV DEBIAN_FRONTEND noninteractive
ARG SHIELD_VERSION
ENV SHIELD_VERSION ${SHIELD_VERSION:-v0.9.0}

RUN apt-get update && apt-get install -y wget

RUN wget https://github.com/starkandwayne/shield/releases/download/${SHIELD_VERSION}/shield-server-linux-amd64.tar.gz
RUN tar xzf shield-server-linux-amd64.tar.gz

RUN cp /shield-server-linux-amd64/agent/shield-agent bin/
RUN cp /shield-server-linux-amd64/cli/shield bin/
RUN cp /shield-server-linux-amd64/daemon/shieldd bin/
RUN cp /shield-server-linux-amd64/daemon/shield-schema bin/
RUN cp /shield-server-linux-amd64/daemon/shield-pipe bin/

# WebUI
RUN mkdir -p /usr/share/shield/webui
RUN cp -R /shield-server-linux-amd64/webui/* /usr/share/shield/webui

COPY ./daemon/shieldd.conf .
COPY ./id_rsa .
COPY ./daemon/wait-for-it.sh .
COPY ./daemon/start-shield.sh .

EXPOSE 8080

CMD ./start-shield.sh
