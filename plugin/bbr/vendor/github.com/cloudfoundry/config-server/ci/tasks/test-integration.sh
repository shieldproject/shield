#!/bin/sh
set -e -x

export GOPATH=$(pwd)
export PATH=/usr/local/go/bin:$GOPATH/bin:$PATH

echo "Starting $DB..."
case "$DB" in
  mysql)
    mv /var/lib/mysql /var/lib/mysql-src
    mkdir /var/lib/mysql
    mount -t tmpfs -o size=256M tmpfs /var/lib/mysql
    mv /var/lib/mysql-src/* /var/lib/mysql/

    sudo service mysql start
    ;;
  postgresql)
    export PATH=/usr/lib/postgresql/9.4/bin:$PATH

    mkdir /tmp/postgres
    mount -t tmpfs -o size=256M tmpfs /tmp/postgres
    mkdir /tmp/postgres/data
    chown postgres:postgres /tmp/postgres/data

    su postgres -c '
      export PATH=/usr/lib/postgresql/9.4/bin:$PATH
      export PGDATA=/tmp/postgres/data
      export PGLOGS=/tmp/log/postgres
      mkdir -p $PGDATA
      mkdir -p $PGLOGS
      initdb -U postgres -D $PGDATA
      pg_ctl start -w -l $PGLOGS/server.log -o "-N 400"
    '
    ;;
  memory)
    echo "Memory DB Noop"
    ;;
  *)
    echo "Usage: DB={mysql|postgresql|memory} $0 {commands}"
    exit 1
esac

go clean -r github.com/cloudfoundry/config-server

go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega

cd src/github.com/cloudfoundry/config-server

bin/test-integration $DB