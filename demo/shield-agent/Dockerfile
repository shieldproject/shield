FROM ubuntu
RUN apt-get update && apt-get -y upgrade \
    && DEBIAN_FRONTEND=noninteractive \
        apt-get install -y bzip2 mysql-client postgresql-client \
    && rm -rf /var/lib/apt/lists/*
COPY in /shield
