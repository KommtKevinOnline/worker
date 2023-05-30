/// <reference no-default-lib="true" />
/// <reference lib="dom" />
/// <reference lib="dom.iterable" />
/// <reference lib="dom.asynciterable" />
/// <reference lib="deno.ns" />

import "$std/dotenv/load.ts";

import { start } from "$fresh/server.ts";
import manifest from "./fresh.gen.ts";

import twindPlugin from "$fresh/plugins/twind.ts";
import twindConfig from "./twind.config.ts";

import { Cron } from "https://deno.land/x/croner@6.0.3/dist/croner.js";
import { PollNewVods } from "./src/CheckVods.ts";

new Cron("0 0 * * * *", () => { PollNewVods('50985620') });

PollNewVods('50985620')

await start(manifest, { plugins: [twindPlugin(twindConfig)] });
