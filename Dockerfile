FROM denoland/deno:alpine

RUN apk add ffmpeg

WORKDIR /app

COPY deps.ts .
RUN deno cache deps.ts

ADD . .

RUN deno cache main.ts

CMD ["run", "--allow-all", "main.ts"]