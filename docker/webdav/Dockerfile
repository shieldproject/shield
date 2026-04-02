FROM nginx:stable-bookworm

COPY nginx.conf /etc/nginx/
COPY webdav /

CMD ["/webdav", "-g", "daemon off;"]
