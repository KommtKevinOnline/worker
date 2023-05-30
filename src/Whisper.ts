
import { Logger, basename, dirname } from "../deps.ts";
import { Transcriber } from "./interfaces/Transcriber.ts";

export class Whisper implements Transcriber {
  constructor(
    private readonly logger: Logger
  ) { }

  public async transcribe(path: string): Promise<string> {
    this.logger.info('Transcribing with Whisper.')
    const t0 = performance.now();

    const whisper = new Deno.Command('whisper', {
      cwd: dirname(path),
      args: [
        basename(path),
        "--language",
        "German",
        "--output_format",
        "json"
      ],
    });

    const { code, stdout, stderr } = await whisper.output();

    console.log(new TextDecoder().decode(stdout))

    const t1 = performance.now();
    this.logger.info(`Transcribing done. Took ${t1 - t0}ms`)

    if (code === 0) {
      const jsonPath = `${dirname(path)}/${basename(path, '.mp3')}.json`

      const transcript = await Deno.readTextFile(jsonPath)

      return transcript
    } else {
      this.logger.error(new TextDecoder().decode(stderr))
    }

    throw new Error('Whisper failed.')
  }
}