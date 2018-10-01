FROM shield:8.0.16
RUN apt-get install -y -q mariadb-client
WORKDIR /shield
ADD agent /shield
ENV PATH="/shield/bin:${PATH}"
CMD ["/shield/src/agent/shield-agent", "-c", "/shield/agent.conf", "-l", "debug"]
