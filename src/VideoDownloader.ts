import { ChunkDownloader } from "./ChunkDownloader.ts";
import { M3U8Parser } from "./M3U8Parser.ts";
import { TwitchApi } from "./Twitch.ts";
import { Video } from "./types/Video.ts";
import { Logger, ProgressBar } from "../deps.ts";

export class VideoDownloader {
  constructor(
    private readonly twitchApi: TwitchApi,
    private readonly m3U8Parser: M3U8Parser,
    private readonly chunkdDownloader: ChunkDownloader,
    private readonly logger: Logger,
  ) { }

  public async download(video: Video): Promise<string> {
    const vodFileName = `./data/${video.id}/vod.ts`;
    const file = await Deno.create(vodFileName);

    const vodCredentials = await this.twitchApi.fetchVodCredentials(video.id);

    const m3u8 = await this.twitchApi.getVodM3u8(video.id, vodCredentials);

    const manifest = this.m3U8Parser.parse(m3u8);
    const bestPlaylist = manifest.playlists[0];

    let playlistBaseURL: string | string[] = bestPlaylist.uri.split("/");
    (playlistBaseURL as string[]).pop();
    playlistBaseURL = (playlistBaseURL as string[]).join("/");

    const playlist = await this.twitchApi.fetchPlaylist(bestPlaylist.uri);
    const segments = playlist.segments;

    // I only want to fetch the latest 20 segments (if there are more than 20)
    const totalSegments = segments.length <= 20 ? segments.length - 1 : segments.length - 20;

    const progress = new ProgressBar({
      title: `Downloading VOD ${video.id}`,
      total: totalSegments,
    });

    for (let i = totalSegments; i < segments.length; i++) {
      const segment = segments[i];

      await this.chunkdDownloader.download(file, `${playlistBaseURL}/${segment.uri}`);
      progress.render(i);
    }

    this.logger.info("Successfully downloaded vod");

    return vodFileName
  }
}