import "https://deno.land/std@0.192.0/dotenv/load.ts";

import { Cron } from "https://deno.land/x/croner@6.0.3/dist/croner.js";
import { PollNewVods } from "./src/CheckVods.ts";

new Cron("0 0 * * * *", () => { PollNewVods('50985620') });

// PollNewVods('50985620')