export interface Transcriber {
  transcribe(audio: string): Promise<string>;
}