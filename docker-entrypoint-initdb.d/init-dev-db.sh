#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  CREATE TABLE IF NOT EXISTS "public"."streams" (
    "date" timestamptz DEFAULT now()
  );

  CREATE TABLE IF NOT EXISTS "public"."vods" (
    "transcript" text,
    "vodid" varchar NOT NULL,
    "title" text DEFAULT ''::text,
    "date" timestamptz DEFAULT now(),
    "url" text DEFAULT ''::text,
    "thumbnail" text DEFAULT ''::text,
    "view_count" int8 DEFAULT 0,
    "online_intend_date" text DEFAULT ''::text,
    "duration" float4 DEFAULT 0,
    PRIMARY KEY ("vodid")
  );
EOSQL
