import { Classifier } from "./interfaces/Classifier.ts";

export class RegexClassifier implements Classifier {
  private REGEX = /\b[mM]orgen(?!.*\bnicht\b)(?=.*(fünfzehn Uhr|wieder))/gmi;

  /**
   * function is async to satisfy the interface.
   * It is possible that future implementations of the Classifier interface will be async.
  */
  // deno-lint-ignore require-await
  public async decide(text: string, streamDate: Date): Promise<Date | false> {
    const matches = [...text.matchAll(this.REGEX)];

    if (matches.length > 0) {
      streamDate.setDate(streamDate.getDate() + 1)
      streamDate.setHours(15, 0, 0, 0)

      return streamDate
    }

    return false;
  }
}
