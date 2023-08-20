export interface Transcriber {
  transcribe(audio: Uint8Array): Promise<string>;
}