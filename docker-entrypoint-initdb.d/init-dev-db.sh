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

  CREATE TABLE "vods" (
      "transcript" text,
      "vodid" varchar NOT NULL,
      PRIMARY KEY ("vodid")
  );

EOSQL