# Example from https://hub.docker.com/_/golang

FROM golang:1.24

WORKDIR /usr/src/app

RUN apt update && apt install -y ffmpeg

# Install https://github.com/cosmtrek/air/
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

CMD ["bash", "/usr/src/app/run_prd.sh"]