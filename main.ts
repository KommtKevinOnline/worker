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
import { Queue } from "./src/Queue.ts";
import { Video } from "./src/types/Video.ts";

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
  classifier: Classifier,
  queue: Queue<Video>,
) {
  const STREAMER_ID = Deno.env.get('STREAMER_ID')

  if (!STREAMER_ID) throw new Error('STREAMER_ID Env Variable not set.')

  const processedVideos = (await client.queryObject<{ vodid: string }>('SELECT vodid FROM vods')).rows.map(row => row.vodid)

  const videos = await twitch.getVideos(STREAMER_ID)

  queue.add(...videos);

  await queue.process(async (video: Video) => {

    if (processedVideos.includes(video.id)) {
      logger.info(`Vod "${video.id}" already processed. Skipping.`);
      return;
    }

    logger.info(`Found new vod "${video.id}" from "${new Date(video.created_at).toLocaleDateString('de-DE')}".`);

    await Deno.mkdir(`./data/${video.id}`, { recursive: true });

    const videoPath = await videoDownloader.download(video)

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
      args: {
        vodid: video.id,
        transcript,
        title: video.title,
        date: video.created_at,
        url: video.url,
        thumbnail: video.thumbnail_url,
        view_count: video.view_count
      },
      text: "INSERT INTO vods (vodId, transcript, title, date, url, thumbnail, view_count) VALUES ($VODID, $TRANSCRIPT, $TITLE, $DATE, $URL, $THUMBNAIL, $VIEW_COUNT)",
    });
  })
}

const m3U8Parser = new M3U8Parser()
const twitchApi = new TwitchApi(m3U8Parser)
const chunkdDownloader = new ChunkDownloader()
const videoDownloader = new VideoDownloader(twitchApi, m3U8Parser, chunkdDownloader, logger)
const audioConverter = new AudioConverter(logger)
const witAi = new WitAi(logger)
const regexClassifier = new RegexClassifier()
const queue = new Queue<Video>(logger)

async function run() {
  await pollVods(twitchApi, videoDownloader, audioConverter, witAi, regexClassifier, queue);
}

new Cron("0 0 * * * *", run);

await run();
