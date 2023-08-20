import * as m3u8Parser from "../deps.ts";
import { Manifest } from "./types/Manifest.ts";

export class M3U8Parser {
  public parse(m3uText: string): Manifest {
    const parser = new m3u8Parser.Parser();
    parser.push(m3uText);
    parser.end();
    return parser.manifest as unknown as Manifest;
  }
}
