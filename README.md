# KommtKevinOnline.de Worker
This is the worker for kommtkevinonline.de
The worker runs each hour and downloads new vods if found.
After the download it gets transcribed with openai whisper.
The result together with an propmt goes to ChatGPT which then decides if Papaplatte has intend of coming online.
ChatGPT returns a json, the json and the vod info from twitch will be written to the database.

### Usage
1. Copy the .env.example to .env
2. Insert missing values
3. Start the project:
```
docker-compose up
```

### How does it work
![image](docs/flow.png)