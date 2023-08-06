#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  DROP TABLE IF EXISTS "public"."alerts";

  CREATE SEQUENCE IF NOT EXISTS alerts_id_seq;

  CREATE TABLE "public"."alerts" (
      "id" int4 NOT NULL DEFAULT nextval('alerts_id_seq'::regclass),
      "title" varchar,
      "text" text,
      "active" bool DEFAULT false,
      "createdAt" timestamp NOT NULL DEFAULT now()
  );

  DROP TABLE IF EXISTS "public"."upcoming_streams";

  CREATE SEQUENCE IF NOT EXISTS upcoming_streams_id_seq;

  CREATE TABLE "public"."upcoming_streams" (
      "id" int4 NOT NULL DEFAULT nextval('upcoming_streams_id_seq'::regclass),
      "date" timestamptz,
      "clip_id" varchar NOT NULL DEFAULT ''::character varying,
      "content" text NOT NULL DEFAULT ''::text
  );

  DROP TABLE IF EXISTS "public"."vods";

  CREATE TABLE "public"."vods" (
      "transcript" text,
      "vodid" varchar NOT NULL,
      PRIMARY KEY ("vodid")
  );

EOSQL