import "https://deno.land/std@0.192.0/dotenv/load.ts";
import { Cron } from "./deps.ts";
import { Client } from "./deps.ts";
import { RegexClassifier } from "./src/RegexClassifier.ts";
import { Classifier } from "./src/interfaces/Classifier.ts";
import { TwitchApi } from "./src/Twitch.ts";
import { VideoDownloader } from "./src/VideoDownloader.ts";
import { AudioConverter } from "./src/AudioConverter.ts";
import { Transcriber } from "./src/interfaces/Transcriber.ts";
import { WitAi } from "./src/WitAi.ts";
import { M3U8Parser } from "./src/M3U8Parser.ts";
import { ChunkDownloader } from "./src/ChunkDownloader.ts";
import { Logger } from "./deps.ts";

if (!Deno.env.get("POSTGRES_PASSWORD")) {
  throw new Error("POSTGRES_PASSWORD Env Variable not set.");
}

const client = new Client({
  hostname: Deno.env.get("POSTGRES_HOST"),
  port: 5432,
  user: Deno.env.get("POSTGRES_USER"),
  password: Deno.env.get("POSTGRES_PASSWORD") ?? '',
  database: Deno.env.get("POSTGRES_DATABASE"),
  tls: {
    enabled: true
  }
});

await client.connect();

const logger = new Logger()

logger.info("Started. Polling every hour.");

async function pollVods(
  twitch: TwitchApi,
  videoDownloader: VideoDownloader,
  audioConverter: AudioConverter,
  transcriber: Transcriber,
  classifier: Classifier
) {
  const STREAMER_ID = Deno.env.get('STREAMER_ID')

  if (!STREAMER_ID) throw new Error('STREAMER_ID Env Variable not set.')

  const LAST_ID = (await client.queryObject<{ vodid: string }>('SELECT vodid FROM vods ORDER BY vodid DESC LIMIT 1')).rows[0]?.vodid;

  const videos = await twitch.getVideos(STREAMER_ID)
  const latestVideo = videos[0];

  if (latestVideo.id === LAST_ID) {
    logger.info("No new vod found.");
    return;
  }

  logger.info(`Found new vod "${latestVideo.id}" from "${new Date(latestVideo.created_at).toLocaleDateString('de-DE')}".`);

  await Deno.mkdir(`./data/${latestVideo.id}`, { recursive: true });

  const videoPath = await videoDownloader.download(latestVideo)

  const audioPath = await audioConverter.convert(videoPath);

  const audio = await Deno.readFile(audioPath)

  const transcript = await transcriber.transcribe(audio)

  const comesOnline = await classifier.decide(transcript)

  if (comesOnline) {
    await client.queryArray({
      args: { date: comesOnline },
      text: "INSERT INTO upcoming_streams (date) VALUES ($DATE)",
    });
  }

  await client.queryArray({
    args: { vodid: latestVideo.id, transcript },
    text: "INSERT INTO vods (vodId, transcript) VALUES ($VODID, $TRANSCRIPT)",
  });
}

const m3U8Parser = new M3U8Parser()
const twitchApi = new TwitchApi(m3U8Parser)
const chunkdDownloader = new ChunkDownloader()
const videoDownloader = new VideoDownloader(twitchApi, m3U8Parser, chunkdDownloader, logger)
const audioConverter = new AudioConverter(logger)
const witAi = new WitAi(logger)
const regexClassifier = new RegexClassifier()

async function run() {
  await pollVods(twitchApi, videoDownloader, audioConverter, witAi, regexClassifier);
}

new Cron("0 0 * * * *", run);

await run();
