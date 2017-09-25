#!/bin/bash

set -e

source /root/.bashrc
cd /tmp
wget -q https://releases.hashicorp.com/vagrant/1.8.4/vagrant_1.8.4_x86_64.deb
dpkg -i vagrant_1.8.4_x86_64.deb
vagrant plugin install vagrant-aws
rm vagrant_1.8.4_x86_64.deb
