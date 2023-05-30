import { PageProps } from "$fresh/server.ts";
import { Video } from "../src/interfaces/Video.ts";

export default function Home({ data }: PageProps<Video[] | null>) {
  return (
    <>
      <h1>Wie funktioniert das?</h1>
    </>
  );
}
