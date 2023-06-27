import "https://deno.land/std@0.192.0/dotenv/load.ts";
import { readKeypress } from "https://deno.land/x/keypress@0.0.11/mod.ts";
import { Cron } from "https://deno.land/x/croner@6.0.3/dist/croner.js";
import { PollNewVods } from "./src/CheckVods.ts";
import { Client } from "https://deno.land/x/postgres@v0.17.0/mod.ts";

if (!Deno.env.get("POSTGRES_PASSWORD")) {
  throw new Error("POSTGRES_PASSWORD Env Variable not set.");
}

const client = new Client({
  hostname: Deno.env.get("POSTGRES_HOST"),
  port: 5432,
  user: Deno.env.get("POSTGRES_USER"),
  password: Deno.env.get("POSTGRES_PASSWORD") ?? '',
  database: Deno.env.get("POSTGRES_DATABASE"),
  tls: {
    enabled: true
  }
});

await client.connect();

console.log("Started. Polling every hour.");

async function pollVods() {
  const result = await PollNewVods("50985620");

  if (result) {
    const { vodid, transcript } = result;

    await client.queryArray({
      args: { vodid, transcript },
      text: "INSERT INTO vods (vodId, transcript) VALUES ($VODID, $TRANSCRIPT)",
    });
  }
}

new Cron("0 0 * * * *", () => {
  pollVods();
});

for await (const keypress of readKeypress()) {
  if (keypress.key === 'm') {
    console.log("Manual Download")
    pollVods();
  }
}

// PollNewVods('50985620')
