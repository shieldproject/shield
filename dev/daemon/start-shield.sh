#!/bin/sh
pg_uri="postgres://postgres:postgres@pg-ssl:5432/shield"

# make sure pg is ready to accept connections
until pg_isready -h pg-ssl -p 5432 -U postgres
do
  echo "Waiting for postgres at: $pg_uri"
  sleep 2;
done

./bin/shield-schema -t postgres -d $pg_uri
./bin/shieldd -c shieldd.conf --log-level debug
