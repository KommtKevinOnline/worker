import { Options } from "$fresh/plugins/twind.ts";

export default {
  selfURL: import.meta.url,
  theme: {
    extend: {
      backgroundImage: {
        "domo": "url('/domo.webp')",
      },
      backgroundColor: {
        "primary": "#a55eea",
      }
    }
  }
} as Options;
