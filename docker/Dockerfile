FROM ubuntu
RUN apt-get update && apt-get install -y --no-install-recommends -q curl build-essential ca-certificates git net-tools netcat openssh-client
RUN mkdir -p /shield/src && \
    curl -Ls https://github.com/starkandwayne/shield/releases/download/v8.0.16/shield-server-linux-amd64.tar.gz | tar -C /shield/src -xz --strip-components 1
RUN mkdir /shield/bin && \
    find /shield/src -not \( -path /shield/src/plugins -prune \) -not \( -path /shield/src/webui -prune \) -type f -exec cp {} /shield/bin \;
