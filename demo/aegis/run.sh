#!/bin/bash
service mysql start
mysql -uroot -e "CREATE DATABASE osticketdb"
mysql -uroot -e "CREATE USER osticket@localhost IDENTIFIED BY 'password'"
mysql -uroot -e "GRANT ALL PRIVILEGES ON osticketdb.* TO osticket@localhost IDENTIFIED BY 'password'"
mysql -uroot -e "FLUSH PRIVILEGES"
mysql -uroot osticketdb < /root/aegis.dump

service nginx start
service php7.0-fpm start

/shield/rc/shield-agent

exec tail -f /var/log/nginx/access.log
