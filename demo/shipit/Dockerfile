FROM ubuntu:14.04
RUN apt-get update
RUN apt-get -y install curl
RUN apt-get install -y tomcat7
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-server \
 && chown -R mysql:mysql /var/lib/mysql /var/run/mysqld \
 && chmod 777 /var/run/mysqld
RUN curl -L https://github.com/openboxes/openboxes/releases/download/v0.7.22/openboxes.war -o /var/lib/tomcat7/webapps/openboxes.war
ADD openboxes-config.properties /usr/share/tomcat7/.grails/openboxes-config.properties
ADD setenv.sh /usr/share/tomcat7/bin/setenv.sh
RUN chmod +x /usr/share/tomcat7/bin/setenv.sh
ADD run.sh /root/run.sh
RUN chmod +x /root/run.sh
VOLUME ["/var/lib/mysql"]
EXPOSE 8080
CMD ["/root/run.sh"]
