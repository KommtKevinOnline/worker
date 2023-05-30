import { asset, Head } from "$fresh/runtime.ts";
import { AppProps } from "$fresh/src/server/types.ts";

export default function App({ Component }: AppProps) {
  return (
    <html data-custom="data">
      <Head>
        <title>Fresh</title>
        <link rel="stylesheet" href={asset("style.css")} />
      </Head>
      <body class="bg-domo bg-auto bg-center bg-no-repeat h-screen text-white flex justify-items-center items-center">
        <video
          autoPlay
          muted
          loop
          id="video-background"
          src={asset("domo.webm")}
        />
        <Component />
      </body>
    </html>
  );
}
