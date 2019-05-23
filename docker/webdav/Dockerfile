FROM nginx:mainline
MAINTAINER James Hunt <james@niftylogic.com>

ADD nginx.conf /etc/nginx/
ADD webdav /

CMD ["/webdav", "-g", "daemon off;"]
