From nginx:latest
RUN apt update && apt install -y curl
RUN mkdir -p /shield/src && \
    curl -Ls https://github.com/starkandwayne/shield/releases/download/v8.0.16/shield-server-linux-amd64.tar.gz | tar -C /shield/src -xz --strip-components 1
ADD nginx.conf /etc/nginx/nginx.conf
ADD nginx.crt /shield/nginx.crt
ADD nginx.key /shield/nginx.key
CMD ["nginx", "-g", "daemon off;"]
