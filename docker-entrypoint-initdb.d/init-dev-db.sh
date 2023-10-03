#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  CREATE TABLE "alerts" (
      "id" SERIAL PRIMARY KEY,
      "title" varchar,
      "text" text,
      "active" bool DEFAULT false,
      "createdAt" timestamp NOT NULL DEFAULT now()
  );

  CREATE TABLE "upcoming_streams" (
      "id" SERIAL PRIMARY KEY,
      "date" timestamptz,
      "clip_id" varchar NOT NULL DEFAULT ''::character varying,
      "content" text NOT NULL DEFAULT ''::text
  );

  CREATE TABLE "public"."vods" (
    "transcript" text,
    "vodid" varchar NOT NULL,
    "title" text DEFAULT ''::text,
    "date" timestamptz DEFAULT now(),
    "url" text DEFAULT ''::text,
    "thumbnail" text DEFAULT ''::text,
    "view_count" int8 DEFAULT 0,
    "online_intend_date" timestamptz,
    "duration" text DEFAULT ''::text,
    PRIMARY KEY ("vodid")
  );

EOSQL