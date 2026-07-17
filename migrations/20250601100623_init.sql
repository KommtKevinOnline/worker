-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "alerts" (
  "id" SERIAL PRIMARY KEY,
  "title" varchar,
  "text" text,
  "active" bool DEFAULT false,
  "created_at" timestamp NOT NULL DEFAULT current_timestamp
);

CREATE TABLE IF NOT EXISTS "predictions" (
  "id" SERIAL PRIMARY KEY,
  "clip_id" varchar,
  "type" varchar,
  "source" varchar,
  "date" timestamptz,
  "topic" text,
  "event_type" varchar,
  "created_at" timestamp NOT NULL DEFAULT current_timestamp
);

CREATE TABLE IF NOT EXISTS "vods" (
  "transcript" text,
  "vodid" varchar NOT NULL,
  "title" text DEFAULT '',
  "date" timestamptz DEFAULT current_timestamp,
  "url" text DEFAULT '',
  "thumbnail" text DEFAULT '',
  "view_count" int8 DEFAULT 0,
  "online_intend_date" text DEFAULT '',
  "duration" float4 DEFAULT 0,
  PRIMARY KEY ("vodid")
);

CREATE TABLE IF NOT EXISTS "twitch_tokens" (
  "id" SERIAL PRIMARY KEY,
  "access_token" text NOT NULL,
  "refresh_token" text NOT NULL,
  "expires_in" int4 NOT NULL,
  "created_at" timestamp NOT NULL DEFAULT current_timestamp
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE "alerts";

DROP TABLE "vods";

DROP TABLE "predictions";

DROP TABLE "twitch_token";

-- +goose StatementEnd
