import * as m3u8Parser from "https://esm.sh/m3u8-parser@6.2.0";
import { Manifest } from "./interfaces/Manifest.ts";

export class M3U8Parser {
  static parse(m3uText: string): Manifest {
    const parser = new m3u8Parser.Parser();
    parser.push(m3uText);
    parser.end();
    return parser.manifest as unknown as Manifest;
  }
}
