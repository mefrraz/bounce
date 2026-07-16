<p align="center">
  <h1 align="center"> Bounce</h1>
  <p align="center"><strong>Smart Sports Data Proxy</strong> para basquetebol português.</p>
</p>

> **v5.1** — 222 jogos + scores em 2.5s · TugaBasket 188-526 jogadores · WebSocket · Swagger · Binário único 11MB

---

## Instalação

### Binário direto (recomendado para Raspberry Pi)

```bash
git clone https://github.com/mefrraz/bounce.git
cd bounce
go build -o bounce ./cmd/server
BOUNCE_DATA_DIR=./data ./bounce
```

### Docker

```bash
docker run -d --name bounce --restart unless-stopped \
  -p 3001:3001 -v bounce-data:/data \
  ghcr.io/mefrraz/bounce:latest
```

### Docker Compose

```bash
git clone https://github.com/mefrraz/bounce.git
cd bounce
docker compose up -d
```

### Oracle Free Tier / VPS

```bash
git clone https://github.com/mefrraz/bounce.git
cd bounce
go build -o bounce ./cmd/server
BOUNCE_DATA_DIR=/opt/bounce/data ./bounce &
```

### Cross-compile (ex: Windows → Raspberry Pi)

```bash
GOOS=linux GOARCH=arm64 go build -o bounce-arm64 ./cmd/server
scp bounce-arm64 pi@192.168.1.200:~/bounce
# No Pi: chmod +x ~/bounce && BOUNCE_DATA_DIR=/tmp/bdata ~/bounce &
```

> Abre `http://localhost:3001/docs` para a documentação interativa Swagger.
> Abre `http://localhost:3001/app` para a Mini-Dribly.

---

## API

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/api/games?club=119&season=2025/2026` | 222-576 jogos com scores (2.5s) |
| `GET` | `/api/game/{id}` | Ficha completa: equipas, 89-73, 4 períodos |
| `GET` | `/api/standings/{compID}` | Classificação (12 equipas) |
| `GET` | `/api/competitions` | 15 competições dinâmicas |
| `GET` | `/api/athlete/{id}` | Atleta: nome, foto, stats |
| `GET` | `/api/team/{id}` | Plantel da equipa |
| `GET` | `/api/club/{clubID}/teams` | 22 equipas do clube |
| `GET` | `/api/tugabasket/standings?competitionId=ID` | 58 equipas regionais |
| `GET` | `/api/tugabasket/players?competitionId=ID` | 188-526 jogadores (18 campos) |
| `GET` | `/api/tugabasket/teams?competitionId=ID` | Stats agregadas por equipa |
| `GET` | `/api/elo` | Ranking ELO |
| `GET` | `/api/predictions/{gameID}` | 57% home win probability |
| `GET` | `/api/h2h?team_a=X&team_b=Y` | Head-to-head |
| `WS` | `/ws/game/{gameID}` | Tempo real + polling inteligente |
| `GET` | `/health` | `{"status":"ok","version":"v5.1.0"}` |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/docs` | Swagger UI |
| `GET` | `/docs/swagger.json` | OpenAPI spec |
| `GET` | `/test` | Consola de testes |
| `GET` | `/app` | Mini-Dribly |

---

## Funcionalidades

| Área | Features |
|---|---|
| **FPB** | Jogos + scores (get_results), classificações, atletas, clubes, equipas, competições |
| **TugaBasket** | Classificações regionais, stats jogadores (22 campos), stats equipas |
| **Cache** | SQLite com TTL adaptativo (2min hoje, 24h histórico, chain invalidation) |
| **Tempo real** | WebSocket + scheduler com polling 2min quando há espectadores |
| **Rate limit** | 100 req/min por IP |
| **Segurança** | Graceful shutdown (SIGTERM), structured logging (slog) |
| **Docs** | Swagger UI em /docs com todos os endpoints |

---

## Stack

| Camada | Tecnologia |
|---|---|
| Linguagem | Go |
| HTTP router | chi |
| HTML parser | goquery |
| Cache | SQLite (modernc.org/sqlite) |
| WebSocket | gorilla/websocket |
| Docker | Multi-stage Alpine (~15MB) |
| CI/CD | GitHub Actions (multi-arch) |
| SDK | TypeScript (`@dribly/bounce-client`) |

---

## Configuração

| Env | Default | Descrição |
|---|---|---|
| `BOUNCE_PORT` | `3001` | Porta HTTP |
| `BOUNCE_DATA_DIR` | `/data` | Diretório do SQLite |

---

## Client SDK

```bash
npm install @dribly/bounce-client
```

```typescript
import BounceClient from '@dribly/bounce-client'

const bounce = new BounceClient('http://localhost:3001')

// 222 jogos do Benfica 2025/2026 com scores
const games = await bounce.games({ club: 119, season: '2025/2026' })

// Ficha de jogo com períodos
const detail = await bounce.game('413420')

// Tempo real
const stop = bounce.watchGame('413420', (event) => {
  console.log(event.type, event.data)
})
```

---

## Testes

```bash
go test ./... -v   # 12 testes, todos passam
go vet ./...        # zero warnings
```

---

## Licença

GNU **AGPLv3** — [LICENSE](LICENSE)
