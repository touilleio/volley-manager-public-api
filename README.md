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

Pre-built images are published to `ghcr.io/touilleio/volley-manager-public-api` by the GitHub
pipeline on every push to `main` (`latest` tag) and on `v*` version tags. Pin a release with
`ghcr.io/touilleio/volley-manager-public-api:<version>` instead of `latest` if you prefer.

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

# Security and production deployment

This service is a **public, read-only API**: `/upcoming`, `/past`, `/ranking`, `/teams`, the
calendar download, and the MCP endpoint intentionally require no authentication. Deploy it only
with data you are comfortable publishing, and never expose the upstream `API_KEY` (it is used
server-side only and never returned in responses).

The Gin server is hardened for production out of the box:

- **Release mode**: the binary defaults to `GIN_MODE=release` (set `GIN_MODE` explicitly to
  override). Release mode disables Gin's debug logging and route warnings.
- **Timeouts**: the `http.Server` sets `ReadHeaderTimeout` (5s), `ReadTimeout` (15s),
  `WriteTimeout` (30s), and `IdleTimeout` (60s) to bound slow-client (Slowloris-style) attacks.
- **Graceful shutdown**: `SIGINT`/`SIGTERM` drains in-flight requests with a 10s deadline.
- **Trusted proxies**: none are trusted by default, so `X-Forwarded-For` cannot be spoofed.
  If you run behind a reverse proxy, terminate TLS there and — only if you need real client IPs —
  set trusted proxies explicitly in code to your proxy addresses.
- **Security headers**: `X-Content-Type-Options: nosniff` and `Referrer-Policy` are set on every
  response. Content-Security-Policy and frame rules are deployment-specific (the static pages use
  CDN assets and the UI may be embedded); configure them at your reverse proxy if needed.
- **MCP endpoint**: `/mcp` runs statelessly, so clients cannot accumulate server-side sessions.
- **TLS**: the container serves plain HTTP on port 8080. Put a TLS-terminating reverse proxy
  (nginx, Traefik, Caddy, cloud load balancer) in front for any non-localhost deployment.

Keep secrets out of the image build: `.dockerignore` excludes `.env`, Terraform state and cache,
game snapshots, and `.git` from the build context. Rotate credentials if a build context or
builder cache containing them ever left your workstation.

# Match notifications

At every poll, freshly fetched matches are compared with the previous state. Moved matches
(date/time, hall, home/away swap) and status changes produce a `volley.matches.changed` JSON event.
Set `PUBLISH_NEW_GAMES=true` to also publish `volley.matches.new` events for newly detected
matches. Matches that changed while the application was stopped are detected on the first poll after
a restart, thanks to a games snapshot stored in the `data` docker volume.

The application publishes events to an SNS topic, not to an SQS queue. Set `SNS_TOPIC_ARN` and the
publisher credentials (`AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`, optionally `AWS_REGION`) in
your `.env` file. `PUBLISH_NEW_GAMES` defaults to `false`. When `SNS_TOPIC_ARN` is unset,
notifications are disabled.

The Terraform setup in [deployment/](./deployment) creates the topic and one subscribed SQS queue
per suffix. Its default suffixes are `generic-wacli` and `marqueurs`. Get the queue URLs with
`terraform output -json queue_urls`; for example, use
`terraform output -json queue_urls | jq -r '."generic-wacli"'` as `QUEUE_URL` for
`scripts/sqs-whatsapp-notifier.sh`. Consumers poll SQS; they do not read from SNS directly. SNS
subscriptions use `raw_message_delivery = true`, so each SQS message `Body` is the raw event JSON
shown below, not an SNS envelope.

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

New-match events have the same envelope and match metadata, without a `changes` array:

```json
{
  "version": 1,
  "type": "volley.matches.new",
  "detectedAt": "2026-07-24T16:00:00+02:00",
  "games": [
    {
      "gameId": 391648,
      "playDate": "2025-09-28 15:00:00",
      "homeTeam": "Gibloux Volley F1",
      "awayTeam": "VBC Sense",
      "league": "2L",
      "hall": "Gymnase, Bulle"
    }
  ]
}
```

# MCP server

An [MCP](https://modelcontextprotocol.io) endpoint is exposed at `/mcp` (stateless Streamable
HTTP), so AI assistants can query the club's matches directly. The server implements MCP
`2026-07-28` with the official Go SDK v1.7.0 and automatically negotiates supported older protocol
versions. Four tools are available; calendar-week tools use Monday to Sunday (Europe/Zurich):

- `get_next_week_upcoming_matches` — matches of next week
- `get_current_week_match_results` — match results of the current week, from Monday up to now
- `list_all_teams` — all managed teams with their `teamId`
- `get_upcoming_matches_for_team` — all future matches for a `teamId` returned by `list_all_teams`
