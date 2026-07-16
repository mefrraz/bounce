# Bounce API — Referência Completa

> Proxy inteligente de dados desportivos para basquetebol português.
> Base URL: `http://localhost:3001`
> **v2.0** — 222 jogos + scores em 2.5s via `get_results`

---

## Índice

1. [Jogos](#jogos)
2. [Ficha de Jogo](#ficha-de-jogo)
3. [Classificações](#classificações)
4. [Atleta](#atleta)
5. [Equipa](#equipa)
6. [Clube](#clube)
7. [Competições](#competições)
8. [TugaBasket](#tugabasket)
9. [ELO & Previsões](#elo--previsões)
10. [WebSocket](#websocket)
11. [Admin](#admin)

---

## Jogos

### `GET /api/games?club=ID&season=YYYY/YYYY`

Jogos de um clube numa época **com scores**. Um pedido = todos os jogos.

| Parâmetro | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| `club` | integer | ✅ | ID do clube (ex: 119=Benfica, 127=Porto, 169=Sporting) |
| `season` | string | ✅ | Época no formato `YYYY/YYYY`. Disponível desde `2003/2004`. |

**Tempo:** ~2.5s (1 pedido HTTP ao `get_results` da FPB)
**Cache:** SQLite permanente após primeiro pedido

```bash
curl "http://localhost:3001/api/games?club=119&season=2025/2026"
```

**Resposta (222 jogos):**
```json
[
  {
    "id": "414520",
    "data": "18 out 2025",
    "hora": "15:00",
    "equipa_casa": "FC Gaia",
    "equipa_fora": "A Indicar",
    "resultado_casa": null,
    "resultado_fora": null,
    "local": "Pavilhão Futebol Clube Gaia",
    "competicao": "1ª Divisão Masculina",
    "escalao": "Sénior masculino",
    "estado": "AGENDADO",
    "logo_casa": "https://sav2.fpb.pt/uploads/clubes/logotipo/...",
    "logo_fora": "https://sav2.fpb.pt/uploads/clubes/logotipo/..."
  }
]
```

---

## Ficha de Jogo

### `GET /api/game/{internalID}`

Detalhe completo de um jogo: equipas, score, logos, períodos.

```bash
curl "http://localhost:3001/api/game/413420"
```

**Resposta:**
```json
{
  "id": "413420",
  "equipa_casa": "SL Benfica",
  "equipa_fora": "Futebol Clube do Porto",
  "resultado_casa": 89,
  "resultado_fora": 73,
  "estado": "FINALIZADO",
  "logo_casa": "https://sav2.fpb.pt/old_uploads/CLU/CLU_127_LOGO.png",
  "logo_fora": "https://sav2.fpb.pt/old_uploads/CLU/CLU_120_LOGO.png",
  "periodos": [
    {"periodo": 1, "casa": 21, "fora": 18},
    {"periodo": 2, "casa": 24, "fora": 19},
    {"periodo": 3, "casa": 22, "fora": 20},
    {"periodo": 4, "casa": 22, "fora": 16}
  ]
}
```

---

## Classificações

### `GET /api/standings/{compID}`

Tabela classificativa de uma competição.

```bash
curl "http://localhost:3001/api/standings/10902"
```

---

## Atleta

### `GET /api/athlete/{id}`

Perfil de atleta: nome, foto, posição, clube, stats.

```bash
curl "http://localhost:3001/api/athlete/269564"
```

---

## Equipa

### `GET /api/team/{id}`

Detalhe da equipa: plantel, jogos, fotos.

```bash
curl "http://localhost:3001/api/team/equipa_57682"
```

---

## Clube

### `GET /api/club/{clubID}/teams`

Lista de equipas de um clube.

```bash
curl "http://localhost:3001/api/club/127/teams"
```

---

## Competições

### `GET /api/competitions`

Lista de competições ativas da FPB (obtidas dinamicamente).

```bash
curl "http://localhost:3001/api/competitions"
```

---

## TugaBasket

### `GET /api/tugabasket/standings?competitionId=ID`

Classificações do TugaBasket com scores.

```bash
curl "http://localhost:3001/api/tugabasket/standings?competitionId=1"
```

---

## ELO & Previsões

### `GET /api/elo`
Ranking ELO nacional.

### `GET /api/predictions/{gameID}`
Probabilidade de vitória baseada em ELO.

### `GET /api/h2h?team_a=X&team_b=Y`
Histórico de confrontos entre duas equipas.

---

## WebSocket

### `WS /ws/game/{gameID}`

Atualizações em tempo real de scores.

```javascript
const ws = new WebSocket("ws://localhost:3001/ws/game/413420")
ws.onmessage = (msg) => {
  const event = JSON.parse(msg.data)
  // { type: "score_update", data: { ... } }
}
```

Eventos: `score_update`, `game_started`, `game_finished`.

---

## Admin

### `GET /health`
Estado do servidor.

### `GET /test`
Consola de testes interativa.

### `GET /app`
Mini-Dribly — interface web para navegar pelos dados.

---

## Notas

- **Cache**: SQLite com TTL inteligente (2min jogos ao vivo, 1h histórico, 24h épocas passadas)
- **Rate limit**: 1 pedido/segundo à FPB
- **Headers**: `User-Agent: Bounce/1.6`, `Referer: https://www.fpb.pt/`
- **Épocas**: desde 2003/2004 até à atual
- **Formato**: JSON em todos os endpoints
