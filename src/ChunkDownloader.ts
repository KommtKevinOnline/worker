import { Buffer } from "https://deno.land/std@0.177.1/io/buffer.ts";

export class ChunkDownloader {
  static async download(writer: Deno.FsFile, downloadUrl: string) {
    const res = await fetch(downloadUrl, { method: "GET" });
    const blob = await res.arrayBuffer();
    writer.write(new Buffer(blob).bytes());
  }
}
