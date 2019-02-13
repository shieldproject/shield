FROM shieldproject/shield-agent
RUN apt-get update
RUN apt-get install -y software-properties-common
RUN add-apt-repository -y ppa:ondrej/php
RUN apt-get update && apt-get -y upgrade \
    && DEBIAN_FRONTEND=noninteractive \
        apt-get install -y \
          nginx \
          php7.0-cli php7.0-mysql php7.0-cgi \
          php7.0-fpm php7.0-gd php7.0-imap \
          php7.0-xml php7.0-mbstring php7.0-intl \
          php-apcu wget \
          mysql-server \
          unzip \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir /var/www/html/osticket \
    && wget --quiet -P /tmp/osticket/ http://osticket.com/sites/default/files/download/osTicket-v1.10.zip \
    && unzip /tmp/osticket/osTicket-v1.10.zip -d /var/www/html/osticket/ \
    && cp /var/www/html/osticket/upload/include/ost-sampleconfig.php /var/www/html/osticket/upload/include/ost-config.php \
    && chown -R www-data:www-data /var/www/html/osticket

ADD osticket /etc/nginx/sites-available/osticket
ADD ost-config.php /var/www/html/osticket/upload/include/ost-config.php
ADD run.sh /root/run.sh
ADD aegis.dump /root/aegis.dump

RUN chmod +x /root/run.sh \
    && ln -s /etc/nginx/sites-available/osticket /etc/nginx/sites-enabled/ \
    && rm /etc/nginx/sites-enabled/default \
    && chown www-data:www-data /var/www/html/osticket/upload/include/ost-config.php \
    && chmod 644 /var/www/html/osticket/upload/include/ost-config.php

ENV AGENT_NAME aegis-box
VOLUME ["/var/lib/mysql"]
CMD ["/root/run.sh"]
