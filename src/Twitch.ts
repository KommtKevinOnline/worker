import { M3U8Parser } from "./M3U8Parser.ts";
import { Video } from "./interfaces/Video.ts";

export class TwitchApi {
  private static BASE_URL = "https://api.twitch.tv/helix";

  private static async getToken(): Promise<string> {
    const client_id = Deno.env.get("TWITCH_CLIENT_ID");
    const client_secret = Deno.env.get("TWITCH_CLIENT_SECRET");

    if (!client_id || !client_secret) {
      throw new Error("Not all Env Variables are set.");
    }

    const params = new URLSearchParams({
      client_id,
      client_secret,
      grant_type: "client_credentials",
    });

    const res = await fetch(
      `https://id.twitch.tv/oauth2/token?${params.toString()}`,
      { method: "POST" },
    );
    const data = await res.json();

    return data.access_token;
  }

  public static async getVideos(userId: string): Promise<Video[]> {
    const clientId = Deno.env.get("TWITCH_CLIENT_ID");

    if (!clientId) throw new Error("TWITCH_CLIENT_ID Env Variable not set.");

    const params = new URLSearchParams({
      user_id: userId,
    });

    const headers = new Headers({
      "Client-Id": clientId,
      "Authorization": `Bearer ${await this.getToken()}`,
    });

    const res = await fetch(`${this.BASE_URL}/videos?${params.toString()}`, {
      headers,
    });
    const { data } = await res.json();

    if (!data) {
      console.debug(data);
      throw new Error("No data returned from Twitch API.");
    }

    return data.map((video: Video) => {
      return {
        ...video,
        created_at: new Date(video.created_at),
        published_at: new Date(video.published_at),
      };
    });
  }

  public static async fetchVodCredentials(vodId: string) {
    const clientId = Deno.env.get("TWITCH_GQL_CLIENT_ID");

    if (!clientId) {
      throw new Error("TWITCH_GQL_CLIENT_ID Env Variable not set.");
    }

    const headers = new Headers({
      "client-id": clientId,
    });

    const res = await fetch("https://gql.twitch.tv/gql", {
      method: "POST",
      body: JSON.stringify({
        operationName: "PlaybackAccessToken_Template",
        query:
          'query PlaybackAccessToken_Template($login: String!, $isLive: Boolean!, $vodID: ID!, $isVod: Boolean!, $playerType: String!) {  streamPlaybackAccessToken(channelName: $login, params: {platform: "web", playerBackend: "mediaplayer", playerType: $playerType}) @include(if: $isLive) {    value    signature    __typename  }  videoPlaybackAccessToken(id: $vodID, params: {platform: "web", playerBackend: "mediaplayer", playerType: $playerType}) @include(if: $isVod) {    value    signature    __typename  }}',
        variables: {
          isLive: false,
          login: "",
          isVod: true,
          vodID: vodId,
          playerType: "site",
        },
      }),
      headers,
    });

    const { videoPlaybackAccessToken } = (await res.json()).data;

    return {
      token: videoPlaybackAccessToken.value,
      sig: videoPlaybackAccessToken.signature,
    };
  }

  public static async getVodM3u8(
    vodId: string,
    { token, sig }: { token: string; sig: string },
  ) {
    const res = await fetch(
      `https://usher.ttvnw.net/vod/${vodId}.m3u8?allow_source=true&token=${token}&sig=${sig}`,
      { method: "GET" },
    );
    const m3u8Playlist = await res.text();

    return m3u8Playlist;
  }

  public static async fetchPlaylist(url: string) {
    const res = await fetch(url, { method: "GET" });
    const m3u8Playlist = await res.text();

    return M3U8Parser.parse(m3u8Playlist);
  }
}
