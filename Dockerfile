FROM denoland/deno:alpine-1.34.3

RUN apk add ffmpeg

WORKDIR /app

USER deno

COPY deps.ts .
RUN deno cache deps.ts

ADD . .

RUN deno cache main.ts

CMD ["run", "--allow-all", "main.ts"]