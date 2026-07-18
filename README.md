<p align="center">
  <h1 align="center">🏀 Bounce</h1>
  <p align="center"><strong>Smart Sports Data Proxy</strong> para basquetebol português.<br>Dados da FPB e TugaBasket, cache inteligente, dashboard em tempo real.</p>
</p>

> **v7.7.0** — Dashboard ao vivo · TUI mode · Docker · binário · 20+ endpoints · [bounce.dribly.pt](https://bounce.dribly.pt)

![build](https://img.shields.io/github/actions/workflow/status/mefrraz/bounce/ci.yml?branch=main)
![version](https://img.shields.io/github/v/tag/mefrraz/bounce?label=version)
![license](https://img.shields.io/github/license/mefrraz/bounce)
![go](https://img.shields.io/github/go-mod/go-version/mefrraz/bounce)

---

## 🎯 Porquê o Bounce

Aceder a dados de basquetebol português é frustrante. A FPB não tem API pública, o TugaBasket é só HTML. Cada app que precisa destes dados tem de implementar o seu próprio scraping — duplicando esforço, sem cache, inconsistente. O Bounce faz esse trabalho uma vez e expõe tudo como JSON limpo, para toda a gente usar.

- ✅ API JSON para FPB + TugaBasket
- ✅ Cache inteligente (SQLite, TTL adaptativo)
- ✅ Painel de métricas em tempo real
- ✅ Open source, self-hosted ou API pública gratuita

---

## ✨ Funcionalidades

### 🔌 API JSON

| Endpoint | Descrição |
|---|---|
| Jogos com scores | `/api/games?club=119&season=2025/2026` |
| Classificações | `/api/standings/10902` — J, V, D, PM, PS, PTS |
| Atletas | `/api/athlete/269564` — foto, stats, clube |
| Equipas | `/api/team/equipa_57682` — plantel, jogos |
| ELO + Previsões | `/api/elo` + `/api/predictions/{id}` |
| TugaBasket | standings, players (22 stats), teams |
| WebSocket | `/ws/game/{id}` — atualizações em tempo real |

### ⚡ Performance

- Cache SQLite com TTL inteligente (2min jogos do dia, 24h histórico)
- Pre-warming ao arranque
- Gzip automático
- Binário ~12MB, imagem Docker ~15MB

### 🖥️ Interface

| Página | URL | O que faz |
|--------|-----|-----------|
| Dashboard | `/dashboard` | Métricas + gráficos canvas |
| API Docs | `/docs` | Documentação interativa |
| API Test | `/test` | Testar cada endpoint |
| Metrics | `/metrics` | JSON com métricas |

---

## 📦 Instalação

### Opção 1 — Docker (recomendado)
```bash
docker pull ghcr.io/mefrraz/bounce:latest
docker run -d --name bounce --restart unless-stopped \
  -p 3001:3001 -v bounce-data:/data \
  ghcr.io/mefrraz/bounce:latest
```

### Opção 2 — Binário
```bash
curl -L https://github.com/mefrraz/bounce/releases/latest/download/bounce-linux-amd64 -o bounce
chmod +x bounce
BOUNCE_DATA_DIR=./data ./bounce &
```

### Opção 3 — Go (from source)
```bash
git clone https://github.com/mefrraz/bounce.git && cd bounce
go build -o bounce ./cmd/server
BOUNCE_DATA_DIR=./data ./bounce &
```

### Verificar
```bash
curl http://localhost:3001/health
# → {"status":"ok","version":"v7.4.20","db_ok":true,"uptime":"5s"}
# Abre http://localhost:3001/dashboard
```

---

## 🎯 Como usar

### API pública (zero instalação)

Aponta os pedidos para `https://bounce.dribly.pt`:

```bash
curl https://bounce.dribly.pt/api/games?club=119&season=2025/2026
curl https://bounce.dribly.pt/api/standings/10902
```

- Sempre online, zero manutenção
- **100 pedidos/minuto** (limite público)
- Ideal para testar ou projetos pequenos

### Self-Hosting (sem limites)

Três opções:

**Docker:**
```bash
docker run -d --name bounce --restart unless-stopped \
  -p 3001:3001 -v bounce-data:/data \
  ghcr.io/mefrraz/bounce:latest
```

**Docker Compose:**
```yaml
services:
  bounce:
    image: ghcr.io/mefrraz/bounce:latest
    restart: unless-stopped
    ports: ["3001:3001"]
    volumes: ["bounce-data:/data"]
```

**Binário:**
```bash
curl -L https://github.com/mefrraz/bounce/releases/latest/download/bounce-linux-amd64 -o bounce
chmod +x bounce && BOUNCE_DATA_DIR=./data ./bounce &
```

### Bypass de rate limit (para o teu site)

```bash
# Site → Bounce (header Origin automático, sem expor chaves)
BOUNCE_TRUSTED_ORIGINS=omeusite.pt ./bounce &

# Backend → Bounce (chave secreta)
DRIBLY_KEY=a-minha-chave ./bounce &
curl -H "X-Dribly-Key: a-minha-chave" http://localhost:3001/api/games?club=119
```

---

## 🎛️ Modos de execução

### Background
```bash
BOUNCE_DATA_DIR=./data ./bounce &
```
Servidor web silencioso. Dashboard em `http://localhost:3001/dashboard`.

### TUI (terminal)
```bash
BOUNCE_TUI=true BOUNCE_DATA_DIR=./data ./bounce
```
Métricas ao vivo no terminal. `r` + Enter = reset. `Ctrl+C` = sair.

---

## ⚙️ Variáveis de Ambiente

| Variável | Default | Descrição |
|----------|---------|-----------|
| `BOUNCE_PORT` | `3001` | Porta HTTP |
| `BOUNCE_DATA_DIR` | `/data` | SQLite + cache TLS |
| `BOUNCE_TUI` | (vazio) | `true` = terminal dashboard |
| `BOUNCE_CORS_ORIGIN` | `*` | Origem CORS |
| `BOUNCE_RATE_LIMIT` | `100` | Pedidos/min por IP |
| `BOUNCE_TLS_DOMAIN` | (vazio) | HTTPS automático (LetsEncrypt) |
| `BOUNCE_TRUSTED_ORIGINS` | (vazio) | Bypass rate limit |
| `DRIBLY_KEY` | (vazio) | Bypass server-to-server |

---

## 🛠️ Stack

| Camada | Tecnologia |
|--------|-----------|
| Linguagem | Go |
| HTTP | chi |
| Cache | SQLite (pure Go, sem CGO) |
| WebSocket | gorilla/websocket |
| HTTPS | autocert (LetsEncrypt) |
| Docker | Multi-stage Alpine (~15MB) |

---

## 📄 Licença

GNU **AGPLv3** — [LICENSE](LICENSE)

<p align="center">
  <a href="https://bounce.dribly.pt">🌐 bounce.dribly.pt</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/mefrraz/bounce">📦 GitHub</a>
</p>
