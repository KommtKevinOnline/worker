import { ffmpeg } from "../deps.ts";

export class AudioConverter {
  public static async convert(path: string) {
    const saveName = `.${path.split(".")[1]}.mp3`;

    await ffmpeg({ ffmpegDir: "/usr/bin/ffmpeg", input: path })
      .audioFilters({
        filterName: "silenceremove",
        options: {
          window: 0,
          detection: "peak",
          start_mode: "all",
          stop_mode: "all",
          stop_periods: -1,
          stop_threshold: 0,
        },
      })
      .save(saveName);

    return saveName;
  }
}
