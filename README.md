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

`CLUB_ID` selects every team belonging to that club (for example `906295`). Leave it empty to
include all clubs. `EXCLUDED_TEAMS_ID` is an optional comma-separated list of individual team IDs
to remove from the selection. The former `TEAMS_ID` allow-list is no longer supported.

`CUP_LEAGUE_CATEGORY_IDS` classifies cup competitions by `leagueCategoryId` (default `4`, national
cup). Cup matches remain visible in upcoming/past lists and team calendars, where they carry a cup
indicator, but they never replace regular league/group metadata or produce a ranking.

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
match is moved (new date/time, new hall, home/away swap) or its status changes, a JSON payload is
published to an AWS SQS queue. Matches that changed while the application was stopped are detected
on the first poll after a restart, thanks to a games snapshot stored in the `data` docker volume.

The payload is formatting-independent; consumers own the rendering (Telegram message, email, ...):

```json
{
  "version": 1,
  "type": "volley.matches.changed",
  "detectedAt": "2026-07-24T16:00:00+02:00",
  "games": [
    {
      "gameId": 391647,
      "playDate": "2025-09-21 15:00:00",
      "homeTeam": "Gibloux Volley F1",
      "awayTeam": "Volley Fribourg",
      "league": "2L",
      "hall": "Gymnase, Bulle",
      "changes": [
        {"field": "playDate", "old": "2025-09-20 17:00:00", "new": "2025-09-21 15:00:00"},
        {"field": "hall", "old": "Salle du Collège, Fribourg", "new": "Gymnase, Bulle"}
      ]
    }
  ]
}
```

Set `SQS_QUEUE_URL` and the publisher credentials (`AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`,
optionally `AWS_REGION`) in your `.env` file (see [.env-example](./.env-example)). The queue and the
IAM users are created by the terraform setup in [deployment/](./deployment). When unset,
notifications are disabled.

# MCP server

An [MCP](https://modelcontextprotocol.io) endpoint is exposed at `/mcp` (streamable HTTP), so AI
assistants can query the club's matches directly. Two tools are available, using calendar weeks
running Monday to Sunday (Europe/Zurich):

- `get_next_week_upcoming_matches` — matches of next week
- `get_current_week_match_results` — match results of the current week, from Monday up to now
