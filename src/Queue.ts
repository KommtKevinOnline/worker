import { Logger } from "../deps.ts";

export class Queue<T> {
  private queue: T[] = [];

  constructor(
    private logger: Logger
  ) { }

  add(...item: T[]) {
    this.queue.push(...item);
  }

  async process(callback: (item: T) => Promise<void>) {
    while (this.queue.length > 0) {
      const item = this.queue.shift();
      if (item) {
        await callback(item);
      }
    }
  }
}