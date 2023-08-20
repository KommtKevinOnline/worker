import { Classifier } from "./interfaces/Classifier.ts";

export class RegexClassifier implements Classifier {
  private REGEX = /morgen.*(fünfzehn Uhr|wieder)/gmi;

  /**
   * function is async to satisfy the interface.
   * It is possible that future implementations of the Classifier interface will be async.
  */
  // deno-lint-ignore require-await
  public async decide(text: string): Promise<Date | false> {
    const matches = [...text.matchAll(this.REGEX)];

    if (matches.length > 0) {
      const date = new Date();
      date.setDate(date.getDate() + 1)
      date.setHours(15, 0, 0, 0)

      return date
    }

    return false;
  }
}