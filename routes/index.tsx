import { Handlers, PageProps } from "$fresh/server.ts";
import { TwitchApi } from "../src/Twitch.ts";
import { Video } from "../src/interfaces/Video.ts";

export const handler: Handlers<Video[] | null> = {
  async GET(_, ctx) {
    const videos = await TwitchApi.getVideos("50985620");
    return ctx.render(videos);
  },
};

export default function Home({ data }: PageProps<Video[] | null>) {
  return (
    <>
      <div class="p-4 mx-auto max-w-screen-md backdrop-blur-xl">
        <h1 class="text-7xl font-bold custom-text-shadow">
          Kommt Kevin heute Online?
        </h1>
        <p class="my-6">
          {data?.[0].title}
        </p>
      </div>
    </>
  );
}
