import { AudioConverter } from "./AudioConverter.ts";
import { ChunkDownloader } from "./ChunkDownloader.ts";
import { M3U8Parser } from "./M3U8Parser.ts";
import { TwitchApi } from "./Twitch.ts";
import { WitAi } from "./WitAi.ts";
import { Client } from "../deps.ts"

export async function PollNewVods(userId: string, client: Client) {
  const LAST_ID = (await client.queryObject<{ vodid: string }>('SELECT vodid FROM vods ORDER BY vodid DESC LIMIT 1')).rows[0]?.vodid;

  const res = await fetch('https://kommtkevinonline.de/api/twitch/stream/50985620')
  const data = await res.json()
  if (data.type === 'live') {
    console.log("Streamer is live. Skipping download.")
    return;
  }

  const videos = await TwitchApi.getVideos(userId);

  const latestVideo = videos[0];

  if (latestVideo.id === LAST_ID) {
    console.log("No new vod found.");
    return;
  }

  console.log("Found new vod. Downloading...");
  const vodCredentials = await TwitchApi.fetchVodCredentials(latestVideo.id);

  const m3u8 = await TwitchApi.getVodM3u8(latestVideo.id, vodCredentials);

  const manifest = M3U8Parser.parse(m3u8);
  const bestPlaylist = manifest.playlists[0];

  let playlistBaseURL: string | string[] = bestPlaylist.uri.split("/");
  (playlistBaseURL as string[]).pop();
  playlistBaseURL = (playlistBaseURL as string[]).join("/");

  const playlist = await TwitchApi.fetchPlaylist(bestPlaylist.uri);
  const segments = playlist.segments;

  await Deno.mkdir(`./data/${latestVideo.id}`, { recursive: true });

  const vodFileName = `./data/${latestVideo.id}/vod.ts`;
  const file = await Deno.create(vodFileName);
  for (let i = segments.length - 20; i < segments.length; i++) {
    const segment = segments[i];
    await ChunkDownloader.download(file, `${playlistBaseURL}/${segment.uri}`);
  }

  file.close();
  console.log("Successfully downloaded vod. Converting to mp3...");

  const audio = await AudioConverter.convert(vodFileName);

  console.log("Successfully converted to mp3. Speech to Text...");

  const transcript = await WitAi.dictation(await Deno.readFile(audio));

  await Deno.writeTextFile(`./data/${latestVideo.id}/vod.txt`, transcript);

  console.log(`${latestVideo.id} ready.`);

  await Deno.writeTextFile("./data/LAST_ID.txt", latestVideo.id);

  return { transcript, vodid: latestVideo.id };
}
