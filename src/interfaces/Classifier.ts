export interface Classifier {
  decide(text: string): Promise<Date | false>;
}