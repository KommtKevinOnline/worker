export interface Classifier {
  decide(text: string, streamDate: Date): Promise<Date | false>;
}