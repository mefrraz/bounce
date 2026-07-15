# Bounce — Análise Técnica e Bloqueios

## Contexto

O Bounce é um proxy de dados em Go que deve substituir o scraping que a Dribly faz no browser, servindo uma API REST para a Dribly consumir.

## O Problema Central

A FPB (fpb.pt) serve dados de jogos de duas formas:

### 1. Páginas estáticas (HTML no servidor) — ✅ Funciona
- **Calendário**: `/calendario/clube_ID/` → 14 jogos com equipas, logos, pavilhões
- **Ficha de jogo**: `/ficha-de-jogo?internalID=ID` → Nomes das equipas (extraídos do `<title>`)
- **Atleta**: `/atletas/ID/` → Nome, foto, posição, stats
- **WordPress AJAX** (classificações): `admin-ajax.php?action=get_more_fase_regular`

Estes endpoints **funcionam no Bounce** com `goquery` (parser HTML). Não precisam de JavaScript.

### 2. Página de resultados — ❌ Bloqueada
- **Resultados**: `/resultados/clube_ID/?clube=ID&epoca=XXXX/XXXX`

Esta página carrega os jogos **100% via JavaScript (AJAX)**. O HTML inicial não contém dados de jogos — apenas o header e footer do site.

## Evidências

### Teste com curl
```
$ curl "https://www.fpb.pt/resultados/clube_119/?clube=119&epoca=2025/2026" | wc -c
83616 bytes  (apenas header/nav, zero jogos)
```

### Teste com Puppeteer (headless Chromium)
```
URL correta, página carrega (título: "Resultados... Futebol Clube de Gaia")
Tempo de espera: 15 segundos após DOMContentLoaded
Resultado: 0 .game-wrapper-a, 0 .day-wrapper, 0 .results_text
Console: "Pageview limit exceeded: Banner disabled"
```

### Teste com o browser do utilizador
```
333609 bytes  (página completa com jogos)
URL: https://www.fpb.pt/resultados/clube_119/?clube=119&epoca=2025/2026
Mesmo assim: 0 game-wrapper-a no HTML fonte
```

O browser do utilizador tem **sessão real** (cookies de visitas anteriores, GPU, WebGL fingerprint, etc.) que a FPB não bloqueia. O headless browser (primeira visita, sem cookies, sem GPU real) recebe uma página vazia.

## O que a Dribly faz

A Dribly **NÃO faz scraping da página de resultados no servidor**. O scraper da Dribly:
1. Corre no **browser do utilizador** (com sessão real)
2. Usa `cheerio` (jQuery) para parsear o HTML já renderizado
3. Guarda os dados em **Supabase** incrementalmente

A Dribly **nunca dependeu de scraping server-side** para a página de resultados porque isso sempre exigiu um browser real com sessão.

## Soluções Possíveis

### A) Usar Puppeteer/Chromedp numa máquina potente
- Oracle Free Tier (4 cores ARM, 24GB RAM) ou VPS
- Manter sessão/cookies entre pedidos
- Custo: ~30s por página, complexidade de manter browser headless
- Risco: FPB pode continuar a bloquear (anti-bot cada vez mais agressivo)

### B) Bounce como acumulador progressivo
- Scraping diário do calendário (funciona) + ficha de jogo
- Guardar cada jogo em SQLite
- Ao fim de uma época, ter o histórico completo
- Épocas passadas: impossível recuperar (FPB já arquivou)
- Vantagem: zero dependência de browser

### C) Bounce lê do Supabase da Dribly
- A Dribly já tem os dados históricos no Supabase
- Bounce serve como cache/proxy inteligente
- Para novos dados: scraping estático + acumulação
- Vantagem: resolve o problema imediatamente

### D) Híbrido
- Bounce faz scraping estático para tudo o que funciona
- Para resultados históricos: lê do Supabase
- Para resultados novos: acumula via scheduler diário
- Serve API unificada para a Dribly

## Recomendação

**Opção B + D combinadas**: O Bounce deve ser um acumulador que faz scrape diário do que a FPB serve estaticamente (calendário, fichas de jogo, atletas). Para dados históricos que a FPB já arquivou, a Dribly continua a usar o Supabase diretamente. Com o tempo, o Bounce acumula o seu próprio histórico e torna-se a fonte primária.

O problema das épocas passadas é **irrecuperável por scraping** — nem a Dribly consegue recuperá-los hoje (só tem porque guardou durante as épocas).
