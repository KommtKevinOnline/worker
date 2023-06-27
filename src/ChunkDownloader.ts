import { Buffer } from "../deps.ts";

export class ChunkDownloader {
  static async download(writer: Deno.FsFile, downloadUrl: string) {
    const res = await fetch(downloadUrl, { method: "GET" });
    const blob = await res.arrayBuffer();
    writer.write(new Buffer(blob).bytes());
  }
}
