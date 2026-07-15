<p align="center">
  <h1 align="center">🏀 Bounce</h1>
  <p align="center"><strong>Proxy inteligente de dados desportivos</strong> para basquetebol português.</p>
</p>

> **v1.0** — 📡 WebSocket em tempo real · 🏆 ELO Ranking · 🐳 Docker multi-arch · [bounce.dribly.pt](https://bounce.dribly.pt)

---

## 🎯 Porquê o Bounce

A [Dribly](https://dribly.pt) precisava de dados da FPB. Cada utilizador fazia scraping no browser — 14 pedidos HTTP, cache isolado, zero tempo real. O Bounce resolve isto: um servidor central que agrega, faz cache e distribui os dados por uma API limpa, com WebSocket para atualizações ao vivo. Um binário, zero dependências externas, open source.

| Funcionalidade | Bounce | Scraping no browser | API oficial FPB |
|---|---|---|---|
| Cache partilhado | ✅ | ❌ | — |
| Tempo real (WebSocket) | ✅ | ❌ | ❌ |
| Multi-plataforma (Docker) | ✅ | ❌ | — |
| Previsões ELO | ✅ | ❌ | ❌ |
| Push notifications | ✅ | ❌ | ❌ |
| **Open source** | ✅ | ✅ | ❌ |
| **100% gratuito** | ✅ | ✅ | ✅ |

---

## ✨ Funcionalidades

### 🔄 Dados
| Funcionalidade | v | Descrição |
|---|---|---|
| 📅 Jogos e agenda | 0.2 | Proxy JSON da FPB com cache SQLite |
| 🏆 Classificações | 0.2 | Tabelas com J, V, D, PM, PS, DIF, PTS |
| 📊 Ficha de jogo | 0.3 | Períodos Q1-Q4, estatísticas por jogador |
| 🏀 Clubes e equipas | 0.3 | Parser HTML das páginas de clube da FPB |
| 🧹 Scraper HTML | 0.3 | Parser completo do HTML da FPB + WordPress AJAX |

### ⚡ Tempo Real
| Funcionalidade | v | Descrição |
|---|---|---|
| 📡 WebSocket | 0.4 | Atualizações ao vivo por jogo (`ws://.../ws/game/:id`) |
| ⏰ Polling inteligente | 0.4 | Só consulta a FPB durante janelas de jogos ativos |
| 🔔 Push notifications | 0.5 | Web Push (VAPID) para início, score e fim de jogo |

### 🧠 Inteligência
| Funcionalidade | v | Descrição |
|---|---|---|
| 🏆 ELO Ranking | 0.5 | Rating ELO nacional (K=32, home advantage +50) |
| 🔮 Previsões | 0.5 | Probabilidade de vitória baseada em ELO + forma |
| 📜 Head-to-head | 0.5 | Histórico de confrontos entre duas equipas |

### 🛡️ Infraestrutura
| Funcionalidade | v | Descrição |
|---|---|---|
| 🐳 Docker | 0.1 | Imagem multi-arch (<15 MB) para AMD64 e ARM64 |
| 📦 SQLite | 0.2 | Cache local com TTL inteligente por tipo de dado |
| 🧪 Test console | 0.3 | Página `/test` para testar endpoints manualmente |
| 🔄 CI/CD | 1.0 | GitHub Actions: test, build multi-arch, push para GHCR |
| 📦 Client SDK | 1.0 | Pacote npm `@dribly/bounce-client` com tipos TypeScript |

---

## 🚀 Quick Start

```bash
docker run -d --name bounce \
  -p 3001:3001 \
  -v bounce-data:/data \
  ghcr.io/mefrraz/bounce:latest
```

Abre `http://localhost:3001/test` para a consola de testes.

### Build manual

```bash
git clone https://github.com/mefrraz/bounce.git
cd bounce
go build -o bounce ./cmd/server
BOUNCE_DATA_DIR=./data ./bounce
```

---

## 🛠️ Stack

| Camada | Tecnologia |
|---|---|
| Linguagem | Go 1.25 |
| HTTP router | chi |
| HTML parser | goquery |
| Cache | SQLite (modernc.org/sqlite) |
| WebSocket | gorilla/websocket |
| Container | Docker multi-stage Alpine |
| CI/CD | GitHub Actions (multi-arch) |
| SDK | TypeScript (npm) |

---

## ⚙️ Arquitetura

O Bounce é um **único binário Go** que agrega dados da FPB e serve uma API REST + WebSocket.

```
┌─────────────────────────────────────────┐
│                 Bounce                   │
│                                          │
│  ┌──────────┐  ┌──────────┐  ┌────────┐ │
│  │ REST API │  │ WebSocket│  │   ⏰    │ │
│  │  (chi)   │  │(gorilla) │  │Scheduler│ │
│  └────┬─────┘  └──────────┘  └────┬───┘ │
│       │                           │      │
│  ┌────┴─────┐               ┌────┴───┐  │
│  │ FPB Proxy│               │ Cache  │  │
│  │(net/http)│               │(SQLite)│  │
│  └────┬─────┘               └────────┘  │
│       │                                  │
│  ┌────┴─────┐                            │
│  │  Scraper │                            │
│  │(goquery) │                            │
│  └──────────┘                            │
└──────────────┬───────────────────────────┘
               │
     ┌─────────┴─────────┐
     │  FPB.pt           │
     │  sav2.fpb.pt      │
     └───────────────────┘
```

### Fontes de dados

| Fonte | Método | Conteúdo |
|---|---|---|
| **FPB** (`fpb.pt`) | HTML scraping + WordPress AJAX | Clubes, jogos, classificações, estatísticas |
| **FPB API** (`sav2.fpb.pt`) | JSON proxy | Jogos, classificações |

---

## 🏗️ Estrutura do Projeto

```
bounce/
├── cmd/server/main.go              # Entrypoint (chi + wiring)
├── internal/
│   ├── api/                        # Handlers REST + test console
│   │   ├── health.go               #   GET /health
│   │   ├── routes.go               #   /api/games, /api/standings, ...
│   │   ├── insights.go             #   /api/elo, /api/predictions, ...
│   │   ├── test.go + test.html     #   /test (consola interativa)
│   ├── models/models.go            # Tipos: Game, Standing, Competition
│   ├── httpclient/client.go        # HTTP com retry + rate-limit
│   ├── cache/cache.go              # SQLite cache com TTL
│   ├── fpbapi/api.go               # Proxy JSON sav2.fpb.pt
│   ├── scraper/                    # Parser HTML FPB.pt
│   │   ├── scraper.go              #   standings, calendar, game detail
│   │   └── scraper_test.go         #   4 testes
│   ├── scheduler/scheduler.go      # Polling inteligente por janela
│   ├── ws/ws.go                    # WebSocket hub
│   └── insights/elo.go             # ELO engine (K=32)
├── client-sdk/                     # npm @dribly/bounce-client
├── Dockerfile                      # Multi-stage Alpine
├── docker-compose.yml              # Dev
├── docker-compose.prod.yml         # Produção
└── .github/workflows/ci.yml        # CI/CD multi-arch
```

---

## 🧪 Testes

```bash
go test ./... -v
```

| Área | Ficheiro | Testes |
|---|---|---|
| Parser HTML FPB | `scraper_test.go` | 4 |
| Parse de datas PT | `scraper_test.go` | integrado |
| Extração de fases | `scraper_test.go` | integrado |

---

## 🔌 API

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/health` | Estado do servidor |
| `GET` | `/test` | Consola de testes interativa |
| `GET` | `/api/games?date=YYYY-MM-DD` | Jogos de uma data |
| `GET` | `/api/standings/{compID}` | Classificação |
| `GET` | `/api/game/{internalID}` | Ficha de jogo |
| `GET` | `/api/competitions` | Lista de competições |
| `GET` | `/api/elo` | Ranking ELO |
| `GET` | `/api/predictions/{gameID}` | Previsão de jogo |
| `GET` | `/api/h2h?team_a=X&team_b=Y` | Histórico de confrontos |
| `WS` | `/ws/game/{gameID}` | Atualizações em tempo real |

---

## 🤝 Contribuir

1. Escolhe uma [issue](https://github.com/mefrraz/bounce/issues) ou cria uma nova
2. Faz fork, clone, branch
3. `go test ./... -v` tem de passar
4. Abre o PR

---

## 📜 Licença

GNU **AGPLv3** — código aberto, copyleft para serviços web. Vê o ficheiro [LICENSE](LICENSE).

---

<p align="center">
  <a href="https://github.com/mefrraz/bounce">📦 GitHub</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/mefrraz/bounce/pkgs/container/bounce">🐳 Docker</a>
  &nbsp;·&nbsp;
  <a href="https://dribly.pt">🌐 Dribly</a>
</p>
