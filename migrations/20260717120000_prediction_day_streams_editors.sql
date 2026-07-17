-- +goose Up
-- +goose StatementBegin
ALTER TABLE "predictions" ADD COLUMN IF NOT EXISTS "day" date;

UPDATE "predictions"
SET "day" = ("date" AT TIME ZONE 'Europe/Berlin')::date
WHERE "day" IS NULL AND "date" IS NOT NULL;

-- Deduplicate before adding the unique index, keeping the newest row per day
DELETE FROM "predictions" a
USING "predictions" b
WHERE a."day" IS NOT NULL AND a."day" = b."day" AND a."id" < b."id";

CREATE UNIQUE INDEX IF NOT EXISTS "predictions_day_key" ON "predictions" ("day");

CREATE TABLE IF NOT EXISTS "streams" (
  "id" varchar PRIMARY KEY,
  "started_at" timestamptz NOT NULL,
  "ended_at" timestamptz,
  "title" text DEFAULT '',
  "category" text DEFAULT '',
  "created_at" timestamp NOT NULL DEFAULT current_timestamp
);

CREATE TABLE IF NOT EXISTS "editors" (
  "twitch_user_id" varchar PRIMARY KEY,
  "login" varchar NOT NULL,
  "display_name" varchar DEFAULT '',
  "avatar" varchar DEFAULT '',
  "added_by" varchar DEFAULT '',
  "created_at" timestamp NOT NULL DEFAULT current_timestamp
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "editors";
DROP TABLE IF EXISTS "streams";
DROP INDEX IF EXISTS "predictions_day_key";
ALTER TABLE "predictions" DROP COLUMN IF EXISTS "day";
-- +goose StatementEnd
