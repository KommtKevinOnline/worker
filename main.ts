import "https://deno.land/std@0.192.0/dotenv/load.ts";
import { Cron } from "./deps.ts";
import { Client } from "./deps.ts";
import { RegexClassifier } from "./src/RegexClassifier.ts";
import { Classifier } from "./src/interfaces/Classifier.ts";
import { TwitchApi } from "./src/Twitch.ts";
import { VideoDownloader } from "./src/VideoDownloader.ts";
import { AudioConverter } from "./src/AudioConverter.ts";
import { Transcriber } from "./src/interfaces/Transcriber.ts";
import { M3U8Parser } from "./src/M3U8Parser.ts";
import { ChunkDownloader } from "./src/ChunkDownloader.ts";
import { Logger } from "./deps.ts";
import { Queue } from "./src/Queue.ts";
import { Video } from "./src/types/Video.ts";
import { WitAi } from "./src/WitAi.ts";
// import { Whisper } from "./src/Whisper.ts";

if (!Deno.env.get("POSTGRES_PASSWORD")) {
  throw new Error("POSTGRES_PASSWORD Env Variable not set.");
}

const logger = new Logger()

logger.info("Started. Polling every hour.");

async function pollVods(
  client: Client,
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

  await client.connect();

  const latestStream = await twitch.getLatestStream(STREAMER_ID)

  await queue.process(async (video: Video) => {

    if (processedVideos.includes(video.id)) {
      logger.info(`Vod "${video.id}" already processed. Skipping.`);
      return;
    }

    if (latestStream && latestStream.id === video.stream_id) {
      logger.info(`Vod "${video.id}" is still a live. Skipping.`);
      return;
    }

    logger.info(`Found new vod "${video.id}" from ${new Date(video.created_at).toISOString()}.`);

    await Deno.mkdir(`./data/${video.id}`, { recursive: true });

    const videoPath = await videoDownloader.download(video)

    const audioPath = await audioConverter.convert(videoPath);

    const transcript = await transcriber.transcribe(audioPath);

    const comesOnline = await classifier.decide(transcript, new Date(video.created_at))

    await client.queryArray({
      args: {
        vodid: video.id,
        transcript,
        title: video.title,
        date: video.created_at,
        url: video.url,
        thumbnail: video.thumbnail_url,
        view_count: video.view_count,
        online_intend: !!comesOnline,
        online_intend_date: comesOnline ? comesOnline : null
      },
      text: `INSERT INTO vods (
        vodId,
        transcript,
        title,
        date,
        url,
        thumbnail,
        view_count,
        online_intend,
        online_intend_date
      ) VALUES (
        $VODID,
        $TRANSCRIPT,
        $TITLE,
        $DATE,
        $URL,
        $THUMBNAIL,
        $VIEW_COUNT,
        $ONLINE_INTEND,
        $ONLINE_INTEND_DATE
      )`,
    });
  })

  await client.end();
}

const m3U8Parser = new M3U8Parser()
const twitchApi = new TwitchApi(m3U8Parser)
const chunkdDownloader = new ChunkDownloader()
const videoDownloader = new VideoDownloader(twitchApi, m3U8Parser, chunkdDownloader, logger)
const audioConverter = new AudioConverter(logger)
const transcoder = new WitAi(logger)
const regexClassifier = new RegexClassifier()
const queue = new Queue<Video>(logger)

async function run() {
  const client = new Client({
    hostname: Deno.env.get("POSTGRES_HOST"),
    port: 5432,
    user: Deno.env.get("POSTGRES_USER"),
    password: Deno.env.get("POSTGRES_PASSWORD") ?? '',
    database: Deno.env.get("POSTGRES_DATABASE"),
    tls: {
      enabled: true
    },
    connection: {
      attempts: 3
    }
  });

  await pollVods(client, twitchApi, videoDownloader, audioConverter, transcoder, regexClassifier, queue);
}


try {
  new Cron("0 0 * * * *", run);

  await run();
} catch (error) {
  logger.error(error)
}

