Volley Manager Public API
====

This project aims at displaying SwissVolley teams and clubs information such as list of matches, results and team ranking
directly from Volley Manager API but without the need of sharing the API key.
The project is packaged as a standalone container, exposing the public endpoint for easy integration inside club's website or app.

Some meaningful links: 
- Volley manager: [https://volleymanager.volleyball.ch](https://volleymanager.volleyball.ch)
- Volley manager API: [https://swissvolley.docs.apiary.io/#reference/indoor](https://swissvolley.docs.apiary.io/#reference/indoor)
- API key: Administration > Club > Webservice/API, [https://volleymanager.volleyball.ch/sportmanager.indoorvolleyball/clubdata/index](https://volleymanager.volleyball.ch/sportmanager.indoorvolleyball/clubdata/index)

# Usage

Configure .env file base on [.env-example](./.env-example) file, and run the container:

```shell
docker compose up -d
```

## Build it

All the sources are provided in this repo, if you're adventurous you can build it yourself.

```shell
docker compose build
```

## Use it

Open your browser at:

http://localhost:8080/

# Match change notifications

At every poll, the freshly fetched matches are compared with the previous state. When a managed
match is moved (new date/time, new hall, home/away swap) or its status changes, a Telegram
message is sent with the old and new values. Matches that changed while the application was
stopped are detected on the first poll after a restart, thanks to a games snapshot stored in
the `data` docker volume.

To enable it, create a bot with [@BotFather](https://t.me/BotFather), invite it to your
group/chat, and set `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` in your `.env` file
(see [.env-example](./.env-example)). When unset, moved matches are written to the application logs instead.
