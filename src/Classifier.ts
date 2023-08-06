export class Classifier {
  private static REGEX = /morgen.*(fünfzehn Uhr|wieder)/gmi;

  public static decide(text: string) {
    const matches = [...text.matchAll(this.REGEX)];

    return matches.length > 0;
  }
}