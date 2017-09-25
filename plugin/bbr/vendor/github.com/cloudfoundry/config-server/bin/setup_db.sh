#!/bin/bash

set -e

case "$1" in
  mysql)
    echo "Setting up mysql db"
    mysql -uroot -ppassword -e "drop database if exists config_server;"
    mysql -uroot -ppassword -e "create database if not exists config_server;"
    ;;
  postgresql)
    echo "Setting up psql db"
    dropdb --if-exists -U postgres config_server
    createdb -U postgres config_server
    ;;
  memory)
    echo "NO OP"
    ;;
  *)
    echo "Usage: DB={mysql2|postgresql|memory} $0 {commands}"
    exit 1
esac