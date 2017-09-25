FROM ubuntu:14.04

RUN locale-gen en_US.UTF-8
RUN dpkg-reconfigure locales
ENV LANG en_US.UTF-8
ENV LC_ALL en_US.UTF-8

#basic deps
RUN apt-get update
RUN apt-get -y upgrade && apt-get clean
RUN apt-get install -y \
	build-essential \
	git \
	mercurial \
	curl \
	wget \
	tar \
	libssl-dev \
	libreadline-dev \
	dnsutils \
	xvfb \
	&& apt-get clean

#global - ruby will need these
# Nokogiri dependencies
RUN apt-get install -y libxslt-dev libxml2-dev && apt-get clean

#global
ADD install-ruby.sh /tmp/install-ruby.sh
RUN chmod a+x /tmp/install-ruby.sh
RUN cd /tmp && ./install-ruby.sh && rm install-ruby.sh

#global
ENV GOROOT /usr/local/go
ENV PATH $GOROOT/bin:$PATH
ADD install-go.sh deps-golang /tmp/
RUN chmod a+x /tmp/install-go.sh
RUN cd /tmp && ./install-go.sh && rm install-go.sh deps-golang

ADD root_bashrc /root/.bashrc

# package manager provides 1.4.3, which is too old for vagrant-aws
ADD install-vagrant.sh /tmp/install-vagrant.sh
RUN /tmp/install-vagrant.sh && rm /tmp/install-vagrant.sh

# lifecycle ssh test
RUN apt-get install -y sshpass && apt-get clean

# integration registry tests
RUN apt-get install -y openssh-server

# for hijack debugging
RUN apt-get install -y lsof psmisc strace
