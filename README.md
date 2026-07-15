# Bounce

Smart Sports Data Proxy — aggregates, caches, and serves basketball data from FPB
with real-time updates and predictive insights. Built for [Dribly](https://dribly.pt).

## Quick Start

```bash
docker run -d --name bounce -p 3001:3001 -v bounce-data:/data ghcr.io/mefrraz/bounce:latest
```

Open `http://localhost:3001/test` for the interactive test console.

## API

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Health check |
| GET | `/test` | Test console |
| GET | `/api/games?date=YYYY-MM-DD&competition=ID` | Games |
| GET | `/api/standings/{compID}` | Standings |
| GET | `/api/game/{internalID}` | Game detail |
| GET | `/api/competitions` | Competition list |
| GET | `/api/elo` | ELO ranking |
| GET | `/api/predictions/{gameID}` | Win probability |
| GET | `/api/h2h?team_a=X&team_b=Y` | Head-to-head |
| WS | `/ws/game/{gameID}` | Real-time updates |

## Client SDK

```bash
npm install @dribly/bounce-client
```

```typescript
import BounceClient from '@dribly/bounce-client'
const bounce = new BounceClient('https://bounce.dribly.pt')
const games = await bounce.games({ date: '2025-06-18' })
```

## Configuration

| Env | Default | Description |
|---|---|---|
| `BOUNCE_PORT` | `3001` | HTTP port |
| `BOUNCE_DATA_DIR` | `/data` | SQLite path |
| `BOUNCE_VAPID_EMAIL` | — | Web Push contact |
| `BOUNCE_VAPID_PRIVATE_KEY` | — | Web Push VAPID key |

## License

AGPLv3
