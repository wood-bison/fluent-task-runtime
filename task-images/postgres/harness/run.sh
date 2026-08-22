#!/usr/bin/env bash
set -Eeuo pipefail

export PGDATA=/tmp/pgdata
export PGHOST=/tmp/pgsocket
export PGUSER=postgres
export PGDATABASE=postgres

mkdir -p "$PGDATA" "$PGHOST"
chown -R postgres:postgres "$PGDATA" "$PGHOST"

gosu postgres initdb --username=postgres --auth=trust --no-locale --encoding=UTF8 "$PGDATA" >/tmp/initdb.log
gosu postgres pg_ctl -D "$PGDATA" -o "-c listen_addresses='' -k $PGHOST" -w start >/tmp/pg-start.log

cleanup() {
  gosu postgres pg_ctl -D "$PGDATA" -m immediate -w stop >/dev/null 2>&1 || true
}
trap cleanup EXIT

psql -X -v ON_ERROR_STOP=1 -f /solution/schema.sql >/tmp/schema.log
bash /hidden-tests/tests/check.sh
