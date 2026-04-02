ARG UBUNTU_RELEASE=noble
ARG GO_VERSION=1.26.1

FROM ubuntu:${UBUNTU_RELEASE} AS build
ARG GO_VERSION
ARG TARGETARCH
ARG VERSION=local

RUN apt-get update \
 && apt-get install -y bzip2 gzip unzip curl git make gcc libc6-dev openssh-client ca-certificates \
 && curl -sL https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz | tar -C /usr/local -xzf - \
 && rm -rf /var/lib/apt/lists/*

ENV PATH="/usr/local/go/bin:/go/bin:${PATH}" \
    GOPATH="/go"

COPY / /go/src/github.com/shieldproject/shield/
RUN cd /go/src/github.com/shieldproject/shield \
 && go mod tidy \
 && go mod vendor \
 && make build BUILD_TYPE="build -ldflags='-X main.Version=$VERSION'"

RUN mkdir -p /dist/bin /dist/plugins /dist/ui \
 && mv /go/src/github.com/shieldproject/shield/shieldd \
       /go/src/github.com/shieldproject/shield/shield-agent \
       /go/src/github.com/shieldproject/shield/shield-crypt \
       /go/src/github.com/shieldproject/shield/shield-report \
       /go/src/github.com/shieldproject/shield/shield-schema \
       /go/src/github.com/shieldproject/shield/bin/shield-pipe \
       /dist/bin/ \
 && for plugin in $(cat /go/src/github.com/shieldproject/shield/plugins); do \
      cp /go/src/github.com/shieldproject/shield/$plugin /dist/plugins/; \
    done \
 && cp -R /go/src/github.com/shieldproject/shield/web/htdocs /dist/ui

# Build shield CLI
RUN cd /go/src/github.com/shieldproject/shield \
 && go build -mod vendor -o /dist/bin/shield ./cmd/shield

# Download Vault
ARG VAULT_VERSION=1.21.4
RUN curl -sLo /tmp/vault.zip https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_${TARGETARCH}.zip \
 && unzip /tmp/vault.zip -d /dist/bin/ \
 && rm /tmp/vault.zip

FROM ubuntu:${UBUNTU_RELEASE}

RUN apt-get update \
 && apt-get install -y curl netcat-openbsd openssh-client \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -r -m -s /bin/bash vcap

COPY --from=build /dist/bin/ /shield/bin/
COPY --from=build /dist/plugins/ /shield/plugins/
COPY --from=build /dist/ui/ /shield/ui/

# Copy init scripts and config
COPY init/core /shield/init/core
COPY init/agent /shield/init/agent
COPY init/shieldd.conf /shield/config/shieldd.conf
COPY init/vault.conf /shield/config/vault.conf

RUN chmod 0755 /shield/init/core /shield/init/agent \
 && mkdir -p /shield/data /shield/vault-data /etc/shield \
 && chown -R vcap:vcap /shield/data /shield/vault-data

ENV PATH="/shield/bin:${PATH}"

EXPOSE 8080 5444
