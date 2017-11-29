#!/bin/bash
chmod 777 /usr/share/tomcat7/.grails/openboxes-config.properties
service mysql start
mysql -uroot -e "CREATE DATABASE openboxes"
mysql -uroot -e "GRANT ALL PRIVILEGES ON * . * TO 'root'@'localhost'"
mysql -uroot -e "SET PASSWORD FOR 'root'@'localhost' = PASSWORD('test')"
mysql -uroot -ptest -e "FLUSH PRIVILEGES"
service tomcat7 start

# The container will run as long as the script is running, that's why
# we need something long-lived here
exec tail -f /var/log/tomcat7/catalina.out
