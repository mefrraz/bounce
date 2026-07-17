<p align="center">
  <h1 align="center">🏀 Bounce</h1>
  <p align="center"><strong>Smart Sports Data Proxy</strong> para basquetebol português.<br>Dados da FPB e TugaBasket, cache inteligente, dashboard em tempo real.</p>
</p>

> **v6.7.0** — HTTPS automático · Health checks · TUI mode · Métricas persistentes · 20 endpoints

---

## 🚀 Quick Start

```bash
git clone https://github.com/mefrraz/bounce.git && cd bounce
go build -o bounce ./cmd/server
BOUNCE_DATA_DIR=./data ./bounce
# Abre http://localhost:3001/dashboard
```

### Docker
```bash
docker run -d --name bounce --restart unless-stopped -p 3001:3001 -v bounce-data:/data ghcr.io/mefrraz/bounce:latest
```

### Raspberry Pi
```bash
GOOS=linux GOARCH=arm64 go build -o bounce ./cmd/server
scp bounce pi@192.168.1.200:~/
# No Pi:
BOUNCE_DATA_DIR=/tmp/bdata ./bounce &
# ou modo TUI (terminal dashboard):
BOUNCE_TUI=true BOUNCE_DATA_DIR=/tmp/bdata ./bounce
```

---

## 🌍 Web Interface

| Página | URL | Descrição |
|--------|-----|-----------|
| **Dashboard** | `/dashboard` | Métricas em tempo real, gráficos, 8 cards |
| **API Docs** | `/docs` | Documentação interativa de todos os endpoints |
| **API Test** | `/test` | Consola para testar cada endpoint no browser |
| **Metrics** | `/metrics` | JSON com todas as métricas do servidor |

---

## 📡 API Endpoints

### Jogos (Games)
| Método | Rota | Parâmetros | Descrição |
|--------|------|-----------|-----------|
| `GET` | `/api/games` | `club`, `season`, `category`, `gender` | Jogos de um clube por época com scores |
| `GET` | `/api/games/today` | — | Jogos agendados para hoje |
| `GET` | `/api/games/live` | — | Jogos a decorrer neste momento |
| `GET` | `/api/games/paginated` | `club`, `season`, `page`, `size` | Jogos com paginação |
| `GET` | `/api/game/{id}` | `id` (game ID) | Ficha completa: equipas, períodos, stats |

### Classificações & Equipas
| Método | Rota | Parâmetros | Descrição |
|--------|------|-----------|-----------|
| `GET` | `/api/standings/{compID}` | `compID` | Classificação de uma competição |
| `GET` | `/api/competitions` | — | Lista de competições disponíveis |
| `GET` | `/api/team/{id}` | `id` (team ID) | Detalhe de uma equipa |
| `GET` | `/api/club/{id}/teams` | `id` (club ID) | Equipas de um clube |

### Atletas
| Método | Rota | Parâmetros | Descrição |
|--------|------|-----------|-----------|
| `GET` | `/api/athlete/{id}` | `id` (athlete ID) | Perfil e estatísticas de um atleta |

### ELO & Previsões
| Método | Rota | Parâmetros | Descrição |
|--------|------|-----------|-----------|
| `GET` | `/api/elo` | — | Ranking ELO de todas as equipas |
| `GET` | `/api/predictions/{gameId}` | `gameId` | Previsão de vencedor baseada em ELO |
| `GET` | `/api/h2h` | `team_a`, `team_b` | Histórico de confrontos directos |

### TugaBasket
| Método | Rota | Parâmetros | Descrição |
|--------|------|-----------|-----------|
| `GET` | `/api/tugabasket/standings` | `competitionId` | Classificação via TugaBasket |
| `GET` | `/api/tugabasket/players` | `competitionId` | Estatísticas de jogadores (22 campos) |
| `GET` | `/api/tugabasket/teams` | `competitionId` | Estatísticas agregadas por equipa |

### Sistema
| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/health` | Health check: status, versão, DB ping, uptime |
| `GET` | `/metrics` | JSON: requests, cache, FPB, goroutines, memória |
| `GET` | `/api/metrics/history?metric=requests&since=1h` | Histórico de uma métrica (delta) |
| `GET` | `/api/metrics/history/simple?minutes=60` | Snapshots completos para o dashboard |
| `WS` | `/ws/game/{id}` | WebSocket: actualizações em tempo real de um jogo |
| `WS` | `/ws/dashboard` | WebSocket: métricas em tempo real para o dashboard |

---

## ⚙️ Environment Variables

| Variável | Default | Descrição |
|----------|---------|-----------|
| `BOUNCE_PORT` | `3001` | Porta HTTP |
| `BOUNCE_DATA_DIR` | `/data` | Diretório para SQLite e cache TLS |
| `BOUNCE_CORS_ORIGIN` | `*` | Origem CORS permitida |
| `BOUNCE_RATE_LIMIT` | `100` | Pedidos/minuto por IP |
| `BOUNCE_LOG_LEVEL` | `warn` | `debug` para logs detalhados |
| `BOUNCE_QUIET` | (vazio) | `true` para silenciar logs de requests |
| `BOUNCE_TUI` | (vazio) | `true` para modo terminal dashboard |
| `BOUNCE_TLS_DOMAIN` | (vazio) | Domínio para HTTPS automático (LetsEncrypt) |
| `BOUNCE_TLS_CACHE` | `$DATA_DIR/autocert` | Diretório de cache dos certificados |
| `BOUNCE_TRUSTED_ORIGINS` | (vazio) | Dominios que bypassam rate limit (separados por virgula) |
| `DRIBLY_KEY` | (vazio) | Chave para bypass do rate limit (header `X-Dribly-Key`) |

### Modos de execução

```bash
# Normal (web server)
./bounce

# Silencioso (sem logs de requests)
BOUNCE_QUIET=true ./bounce

# TUI (terminal dashboard ao vivo)
BOUNCE_TUI=true BOUNCE_DATA_DIR=/tmp/bdata ./bounce

# HTTPS automático (requer domínio público)
BOUNCE_TLS_DOMAIN=api.example.com ./bounce

# Rate limit alto para testes
BOUNCE_RATE_LIMIT=5000 ./bounce
```

---

## 🏠 Self-Hosting Guide

### Para usares o Bounce no teu próprio site

**1. Hospeda o Bounce** (VPS, Raspberry Pi, Oracle Free Tier):
```bash
git clone https://github.com/mefrraz/bounce.git && cd bounce
go build -o bounce ./cmd/server
BOUNCE_DATA_DIR=./data ./bounce &
```

**2. Configura trusted origins** para o teu site bypassar o rate limit:
```bash
BOUNCE_TRUSTED_ORIGINS=omeusite.pt,outrosite.com BOUNCE_DATA_DIR=./data ./bounce &
```
- O teu site pode fazer pedidos ilimitados à API sem rate limit
- **Sem expor API keys no frontend** — o bypass é feito pelo header `Origin` que o browser envia automaticamente
- Todos os outros visitantes ficam limitados a 100 req/min (configurável)

**3. Para server-to-server** (backend → Bounce), usa `X-Dribly-Key`:
```bash
DRIBLY_KEY=a-minha-chave-secreta BOUNCE_DATA_DIR=./data ./bounce &
```
```bash
curl -H "X-Dribly-Key: a-minha-chave-secreta" http://meu-bounce:3001/api/games?club=119
```

### Como funciona o bypass

| Método | Use case | Expõe a chave? |
|--------|----------|---------------|
| `BOUNCE_TRUSTED_ORIGINS` | Site estático / SPA → Bounce | ❌ Não (header Origin automático) |
| `X-Dribly-Key` | Backend → Bounce | ❌ Não (server-side only) |
| Sem bypass | Visitantes públicos | 100 req/min por IP |



---

## 📊 Dashboard

O dashboard em `/dashboard` mostra:

**Métricas (tempo real via WebSocket):**
- Requests totais · Cache hit rate % · FPB requests · Rate limited
- Uptime · Goroutines · Cache misses · Reqs/sec

**Gráficos (snapshots a cada 10s):**
- Requests/min · Cache hit rate % · FPB requests
- Total requests cumulativo (full-width)
- Seletor de tempo: 1m · 5m · 1h · 6h · 24h · 7d
- Refresh: Live (WebSocket) · 5s · 15s · 60s
- Tooltip com hora exacta ao passar o rato

---

## 🧪 Stress Test Scripts

```bash
# Linux/Pi — 200 workers, 120 segundos
./bounce-stress.sh 192.168.1.200:3001 500 120

# Windows PowerShell — 400 workers, modo agressivo
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\bounce-stress.ps1 192.168.1.200:3001 400 120
```

---

## 🛠️ Stack

| Camada | Tecnologia |
|--------|-----------|
| Linguagem | Go |
| HTTP Router | chi |
| Cache | SQLite (modernc.org/sqlite) |
| WebSocket | gorilla/websocket |
| HTTPS | autocert (LetsEncrypt) |
| Docker | Multi-stage Alpine |
| CI/CD | GitHub Actions (multi-arch) |

---

## 📦 Client SDK

```bash
npm install @dribly/bounce-client
```

```typescript
import BounceClient from '@dribly/bounce-client'
const bounce = new BounceClient('http://localhost:3001')
const games = await bounce.games({ club: 119, season: '2025/2026' })
```

---

## 🧪 Testes

```bash
go test ./... -v
go vet ./...
```

---

## 📄 Licença

GNU **AGPLv3** — [LICENSE](LICENSE)
