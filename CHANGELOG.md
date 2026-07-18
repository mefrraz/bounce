# Changelog

## v7.5.7 (2026-07-17)
- Clean repository: removed binaries, old scripts, duplicate docs
- Dashboard restored from v7.4.18 (working charts)
- Badges in README

## v7.5.1
- Dockerfile uses `golang:alpine` (fixes CI build)

## v7.5.0
- New README inspired by Dribly
- Public API at bounce.dribly.pt

## v7.4.x
- TUI mode with keyboard handler (r = reset)
- Dashboard: 4 metric cards + 3 canvas charts + time range selector
- JSON tags on metrics history for correct chart rendering
- Auto-clear charts on reset
- Removed broken WebSocket code from dashboard
- Fixed `prev is not defined` JS error

## v7.0.0
- Initial release
- FPB + TugaBasket proxy
- SQLite cache with TTL
- WebSocket for live game updates
- ELO ranking engine
- Docker + docker-compose support
