#!/bin/bash

cd "${PGDATA}"
cp /etc/ssl/certs/ssl-cert-snakeoil.pem "${PGDATA}"/server.crt
cp /etc/ssl/private/ssl-cert-snakeoil.key "${PGDATA}"/server.key
chmod og-rwx server.key
chown -R postgres:postgres "${PGDATA}"

# turn on ssl
sed -ri "s/^#?(ssl\s*=\s*)\S+/\1'on'/" "$PGDATA/postgresql.conf"
