import { FfmpegClass, Logger, ProgressBar } from "../deps.ts";

export class AudioConverter {
  constructor(
    private readonly logger: Logger
  ) { }

  public async convert(videoPath: string) {
    const saveName = `.${videoPath.split(".")[1]}.mp3`;

    const ffmpegInstance = await new FfmpegClass({
      ffmpegDir: '/usr/bin/ffmpeg',
      input: videoPath,
    })
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
      .save(saveName, true);

    const progressBar = new ProgressBar({
      title: 'Converting VOD to audio',
      total: 100,
    });

    for await (const progress of ffmpegInstance) {
      progressBar.render(progress.percentage)
    }

    this.logger.info("Successfully converted vod to audio");

    return saveName;
  }
}
