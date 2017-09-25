FROM ubuntu:latest

RUN apt-get update && \
    apt-get install -y --no-install-recommends git python-pip zip jq wget && \
    pip install --upgrade pip && \
    apt-get clean
RUN pip install setuptools && pip install awscli

RUN cd /tmp && \
    wget -nv https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz && \
    ( \
        echo '1862f4c3d3907e59b04a757cfda0ea7aa9ef39274af99a784f5be843c80c6772 go1.8.3.linux-amd64.tar.gz' | \
        sha256sum -c - \
    ) && \
    tar -C /usr/local -xzf go*.tar.gz && \
    rm go*.tar.gz

RUN mkdir -p /opt/go
ENV GOPATH /opt/go
ENV PATH /usr/local/go/bin:/opt/go/bin:$PATH

RUN go get github.com/onsi/ginkgo/ginkgo && go get github.com/golang/lint/golint