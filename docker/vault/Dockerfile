FROM ubuntu
RUN apt-get update && apt-get install -y --no-install-recommends -q curl build-essential ca-certificates git net-tools netcat unzip jq
WORKDIR /vault
RUN mkdir -p /vault/bin && \
    cd /vault/bin && \
    curl -o vault.zip -sL https://releases.hashicorp.com/vault/0.9.0/vault_0.9.0_linux_amd64.zip && unzip vault.zip
ENV PATH="/vault/bin:${PATH}"
ENV VAULT_SKIP_VERIFY=true
ADD vault /vault
CMD ["/vault/bin/vault", "server", "-config", "/vault/vault.conf"]
