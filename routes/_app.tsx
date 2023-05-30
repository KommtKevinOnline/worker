import { asset, Head } from "$fresh/runtime.ts";
import { AppProps } from "$fresh/src/server/types.ts";
import Header from "../components/Header.tsx";

export default function App({ Component }: AppProps) {
  return (
    <html data-custom="data">
      <Head>
        <title>Fresh</title>
        <link rel="stylesheet" href={asset("style.css")} />
      </Head>
      <body class="bg-domo bg-auto bg-center bg-no-repeat h-screen w-screen flex-col text-white flex justify-items-center items-center">
        <video
          autoPlay
          muted
          loop
          id="video-background"
          src={asset("domo.webm")}
        />
        <div class="flex flex-col w-screen">
          <Header title="" active="/" />
        </div>
        <Component />
      </body>
    </html>
  );
}
