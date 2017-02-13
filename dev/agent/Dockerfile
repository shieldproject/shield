# Using base pg image for postgres backup/restore to work 
FROM postgres:9.6
ARG SHIELD_VERSION
ENV SHIELD_VERSION ${SHIELD_VERSION:-v0.9.0}

RUN apt-get update && apt-get install -y wget

RUN wget https://github.com/starkandwayne/shield/releases/download/${SHIELD_VERSION}/shield-server-linux-amd64.tar.gz
RUN tar xzf shield-server-linux-amd64.tar.gz

RUN mkdir -p /plugins/bin
RUN mkdir -p /.ssh

RUN cp /shield-server-linux-amd64/plugins/* /plugins/bin/
RUN cp /shield-server-linux-amd64/agent/shield-agent /bin/shield-agent
RUN cp /shield-server-linux-amd64/daemon/shield-pipe /bin/shield-pipe

# MySQL
RUN apt-get install libaio1
RUN apt-get install libnuma1

RUN wget https://dev.mysql.com/get/Downloads/MySQL-5.7/mysql-common_5.7.17-1debian8_amd64.deb
RUN dpkg -i mysql-common_5.7.17-1debian8_amd64.deb

RUN wget https://dev.mysql.com/get/Downloads/MySQL-5.7/mysql-community-client_5.7.17-1debian8_amd64.deb
RUN dpkg -i mysql-community-client_5.7.17-1debian8_amd64.deb

RUN mkdir -p /var/vcap/packages/shield-mysql/bin

# shield agent config
COPY ./agent/shield-agent.conf /

COPY ./id_rsa /.ssh/id_rsa
COPY ./id_rsa.pub /.ssh/authorized_keys

EXPOSE 4222

CMD ["./bin/shield-agent", "-c", "/shield-agent.conf", "--log-level", "debug"]
