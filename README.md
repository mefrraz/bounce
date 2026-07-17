<p align="center">
  <h1 align="center">🏀 Bounce</h1>
  <p align="center"><strong>Smart Sports Data Proxy</strong> para basquetebol português.<br>Dados da FPB e TugaBasket, cache inteligente, dashboard em tempo real.</p>
</p>

> **v7.4.18** — Dashboard ao vivo · TUI mode · Métricas persistentes · 20+ endpoints

---

## 🚀 Self-Hosting — Passo a Passo

### Pré-requisitos
- **Go 1.24+** ou **Docker**
- ~50MB de espaço em disco
- Raspberry Pi, VPS, ou PC local

### Passo 1: Clonar o repositório
```bash
git clone https://github.com/mefrraz/bounce.git
cd bounce
```

### Passo 2: Build
```bash
# Binário nativo
go build -o bounce ./cmd/server

# Ou Docker (multi-arch: amd64 + arm64)
docker build -t bounce .
```

### Passo 3: Criar diretório de dados
```bash
mkdir -p ./data
```

### Passo 4: Escolher o modo e correr

#### Modo 1 — Background (servidor web silencioso)
```bash
BOUNCE_DATA_DIR=./data ./bounce &
# Abre http://localhost:3001/dashboard
```

#### Modo 2 — TUI (terminal dashboard ao vivo)
```bash
BOUNCE_TUI=true BOUNCE_DATA_DIR=./data ./bounce
```
Mostra métricas em tempo real no terminal:
- Requests, Cache %, FPB Reqs, Rate Limited
- Últimas 8 requests em log ao vivo
- `r` + Enter = reset de todas as métricas
- `Ctrl+C` = sair

### Passo 5: Verificar
```bash
curl http://localhost:3001/health
# {"status":"ok","version":"v7.4.18","db_ok":true,"uptime":"5s"}
```

---

## 🐳 Docker
```bash
# Pull da imagem pública
docker pull ghcr.io/mefrraz/bounce:latest

# Correr
docker run -d --name bounce --restart unless-stopped \
  -p 3001:3001 -v bounce-data:/data \
  ghcr.io/mefrraz/bounce:latest

# Ver logs
docker logs bounce
```

---

## 🖥️ Raspberry Pi
```bash
# Compilar no PC para o Pi
GOOS=linux GOARCH=arm64 go build -o bounce ./cmd/server

# Enviar via SCP
scp bounce piserver@192.168.1.200:~/

# No Pi: correr em background
BOUNCE_DATA_DIR=/tmp/bdata ./bounce &
```

---

## 🌍 Web Interface

| Página | URL | Descrição |
|--------|-----|-----------|
| **Dashboard** | `/dashboard` | Métricas em tempo real, gráficos canvas, 8 cards |
| **API Docs** | `/docs` | Documentação interativa de todos os endpoints |
| **API Test** | `/test` | Consola para testar cada endpoint no browser |
| **Metrics** | `/metrics` | JSON com todas as métricas do servidor |

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
| `BOUNCE_TRUSTED_ORIGINS` | (vazio) | Dominios que bypassam rate limit |
| `DRIBLY_KEY` | (vazio) | Chave para bypass do rate limit (header `X-Dribly-Key`) |

### Modos de execução

```bash
# Background — web server
./bounce

# Silencioso — sem logs de requests
BOUNCE_QUIET=true ./bounce

# TUI — terminal dashboard ao vivo
BOUNCE_TUI=true BOUNCE_DATA_DIR=/tmp/bdata ./bounce

# HTTPS automático — domínio público com LetsEncrypt
BOUNCE_TLS_DOMAIN=api.exemplo.com ./bounce

# Debug — logs detalhados
BOUNCE_LOG_LEVEL=debug ./bounce
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

## 📄 Licença

GNU **AGPLv3** — [LICENSE](LICENSE)
