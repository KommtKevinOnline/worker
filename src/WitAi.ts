import { Logger } from "../deps.ts";
import { Transcriber } from "./interfaces/Transcriber.ts";

export class WitAi implements Transcriber {
  private BASE_URL = "https://api.wit.ai";

  constructor(
    private readonly logger: Logger,
  ) { }

  private parseResponse(response: string) {
    const chunks = response
      .split("\r\n")
      .map((x: string) => x.trim())
      .filter((x: string) => x.length > 0);

    let prev = "";
    const jsons = [];
    for (const chunk of chunks) {
      try {
        prev += chunk;
        jsons.push(JSON.parse(prev));
        prev = "";
      } catch (_e) {
        // not a complete json yet
      }
    }

    return jsons;
  }

  public async transcribe(audioPath: string): Promise<string> {
    const accessToken = Deno.env.get("WITAI_ACCESS_TOKEN");

    if (!accessToken) {
      throw new Error("WITAI_ACCESS_TOKEN Env Variable not set.");
    }

    const audio = await Deno.readFile(audioPath)

    const headers = new Headers({
      "Content-Type": "audio/mpeg3",
      "Authorization": `Bearer ${accessToken}`,
    });

    const res = await fetch(`${this.BASE_URL}/dictation`, {
      method: "POST",
      headers,
      body: audio,
    });

    const texts = [];

    const reader = res.body?.getReader();
    const decoder = new TextDecoder();

    let contents = "";

    let chunk = await reader?.read();

    while (!chunk?.done) {
      contents += decoder.decode(chunk?.value, { stream: !chunk?.done });
      chunk = await reader?.read();
    }

    for (const res of this.parseResponse(contents)) {
      const { error, is_final, text } = res;

      if (!error) {
        if (is_final) {
          texts.push(text);
        }
      }
    }

    this.logger.info("Successfully transcribed vod");

    return texts.join("\n");
  }
}
