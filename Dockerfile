# Example from https://hub.docker.com/_/golang

FROM golang:1.24

WORKDIR /usr/src/app

RUN apt update && apt install -y ffmpeg

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

EXPOSE 4090

CMD ["go", "run", "."]