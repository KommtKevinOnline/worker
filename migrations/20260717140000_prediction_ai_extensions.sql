-- +goose Up
-- +goose StatementBegin
ALTER TABLE "predictions" ADD COLUMN IF NOT EXISTS "confidence" real;
ALTER TABLE "predictions" ADD COLUMN IF NOT EXISTS "quote" text DEFAULT '';
ALTER TABLE "predictions" ADD COLUMN IF NOT EXISTS "quote_start" real;

CREATE TABLE IF NOT EXISTS "prediction_runs" (
  "id" SERIAL PRIMARY KEY,
  "source" varchar NOT NULL DEFAULT '',
  "clip_id" varchar NOT NULL DEFAULT '',
  "model" varchar NOT NULL DEFAULT '',
  "input" text NOT NULL DEFAULT '',
  "operations" text NOT NULL DEFAULT '',
  "response" text NOT NULL DEFAULT '',
  "error" text NOT NULL DEFAULT '',
  "created_at" timestamp NOT NULL DEFAULT current_timestamp
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "prediction_runs";
ALTER TABLE "predictions" DROP COLUMN IF EXISTS "quote_start";
ALTER TABLE "predictions" DROP COLUMN IF EXISTS "quote";
ALTER TABLE "predictions" DROP COLUMN IF EXISTS "confidence";
-- +goose StatementEnd
