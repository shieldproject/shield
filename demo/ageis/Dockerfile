FROM shieldproject/shield-agent
RUN apt-get update && apt-get -y upgrade
RUN apt-get install -y nginx \
      php7.0-cli php7.0-mysql php7.0-cgi \
      php7.0-fpm php7.0-gd php7.0-imap \ 
      php7.0-xml php7.0-mbstring php7.0-intl \
      php-apcu wget
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-server
RUN apt-get install -y unzip
RUN mkdir /var/www/html/osticket
RUN wget -P /tmp/osticket/ http://osticket.com/sites/default/files/download/osTicket-v1.10.zip
RUN unzip /tmp/osticket/osTicket-v1.10.zip -d /var/www/html/osticket/
RUN cp /var/www/html/osticket/upload/include/ost-sampleconfig.php /var/www/html/osticket/upload/include/ost-config.php
RUN chown -R www-data:www-data /var/www/html/osticket
ADD osticket /etc/nginx/sites-available/osticket
RUN ln -s /etc/nginx/sites-available/osticket /etc/nginx/sites-enabled/
RUN rm /etc/nginx/sites-enabled/default
ADD run.sh /root/run.sh
RUN chmod +x /root/run.sh
VOLUME ["/var/lib/mysql"]
CMD ["/root/run.sh"]
