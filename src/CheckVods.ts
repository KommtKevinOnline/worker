import { AudioConverter } from "./AudioConverter.ts";
import { ChunkDownloader } from "./ChunkDownloader.ts";
import { M3U8Parser } from "./M3U8Parser.ts";
import { TwitchApi } from "./Twitch.ts";
import { WitAi } from "./WitAi.ts";

const LAST_ID = "1829750881"

export async function PollNewVods (userId: string) {
  const videos = await TwitchApi.getVideos(userId)

  const latestVideo = videos[0]

  if (latestVideo.id !== LAST_ID) {
    console.log("Found new vod. Downloading...")
    const vodCredentials = await TwitchApi.fetchVodCredentials(latestVideo.id)

    const m3u8 = await TwitchApi.getVodM3u8(latestVideo.id, vodCredentials)

    const manifest = M3U8Parser.parse(m3u8);
    const bestPlaylist = manifest.playlists[0];

    let playlistBaseURL: string | string[] = bestPlaylist.uri.split("/");
    (playlistBaseURL as string[]).pop();
    playlistBaseURL = (playlistBaseURL as string[]).join("/");

    const playlist = await TwitchApi.fetchPlaylist(bestPlaylist.uri)
    const segments = playlist.segments;

    await Deno.mkdir(`./data/${latestVideo.id}`);

    const vodFileName = `./data/${latestVideo.id}/vod.ts`
    const file = await Deno.create(vodFileName);
    for (let i = segments.length - 20; i < segments.length; i++) {
      const segment = segments[i];
      await ChunkDownloader.download(file, `${playlistBaseURL}/${segment.uri}`);
    }

    file.close();
    console.log("Successfully downloaded vod. Converting to mp3...")

    const audio = await AudioConverter.convert(vodFileName);

    console.log("Successfully converted to mp3. Speech to Text...")

    const transcript = await WitAi.dictation(await Deno.readFile(audio));

    Deno.writeTextFile(`./data/${latestVideo.id}/vod.txt`, transcript);

    console.log(`${latestVideo.id} ready.`)
  } else {
    console.log('No new vod found.')
  }
}