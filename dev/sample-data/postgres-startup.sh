#!/bin/sh

# make sure pg is ready to accept connections
until pg_isready -h postgres -p 5432 -U postgres
do
  echo "Waiting for postgres at: $pg_uri"
  sleep 2;
done

psql postgres://postgres:postgres@postgres:5432/shield < postgres.sql
