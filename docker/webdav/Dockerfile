FROM nginx:1.31-trixie
MAINTAINER James Hunt <james@niftylogic.com>

ADD nginx.conf /etc/nginx/
ADD webdav /

CMD ["/webdav", "-g", "daemon off;"]
