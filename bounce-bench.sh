#!/bin/bash
# bounce-bench.sh — aggressive benchmark / metrics smoke test
# Usage: ./bounce-bench.sh [host] [concurrency]
#   host defaults to localhost:3001
#   concurrency defaults to 10

HOST="${1:-localhost:3001}"
CONC="${2:-10}"
BASE="http://$HOST"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
pass=0; fail=0; total=0

spinner=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)
spin() { printf "\r${YELLOW}%s${NC}  %3d/%d  pass=%d fail=%d" "${spinner[$((i%10))]}" "$total" "$N" "$pass" "$fail"; }

hit() {
  local method="$1" path="$2" label="$3"
  ((total++))
  if [ "$method" = "GET" ]; then
    code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$BASE$path" 2>/dev/null)
  else
    code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 -X POST "$BASE$path" 2>/dev/null)
  fi
  if [ "$code" = "200" ] || [ "$code" = "302" ]; then ((pass++)); else ((fail++)); fi
  spin
}

echo -e "${GREEN}═══ Bounce Benchmark ═══${NC}"
echo "Host: $BASE   Concurrency: $CONC"
echo ""

# ── phase 1: sequential warm ──
echo -e "${YELLOW}[1/3] Sequential warm-up${NC}"
endpoints=(
  "GET /health health"
  "GET /api/competitions competitions"
  "GET /api/games?club=119 games-benfica"
  "GET /api/games?club=127 games-fcp"
  "GET /api/standings/10902 standings"
  "GET /api/elo elo"
  "GET /api/games/today today"
  "GET /api/tugabasket/standings?competitionId=1 tb-standings"
  "GET /api/tugabasket/players?competitionId=1 tb-players"
  "GET /api/tugabasket/teams?competitionId=1 tb-teams"
)
N=${#endpoints[@]}; i=0; total=0
for ep in "${endpoints[@]}"; do
  read method path label <<< "$ep"; ((i++))
  hit "$method" "$path" "$label"
done
echo ""

# ── phase 2: parallel burst ──
echo -e "${YELLOW}[2/3] Parallel burst ×${CONC}${NC}"
burst_eps=(
  "GET /api/games?club=119"
  "GET /api/games?club=127&season=2025/2026"
  "GET /api/standings/10902"
  "GET /api/competitions"
  "GET /api/elo"
  "GET /api/h2h?team_a=127&team_b=120"
  "GET /api/predictions/413420"
  "GET /api/games/today"
  "GET /api/games/live"
  "GET /health"
)
N=$(( ${#burst_eps[@]} * CONC )); i=0; total=0
for ((r=0; r<CONC; r++)); do
  for ep in "${burst_eps[@]}"; do
    read method path <<< "$ep"; ((i++))
    hit "$method" "$path" "burst" &
  done
done
wait
echo ""

# ── phase 3: rapid single endpoint (rate-limit test) ──
echo -e "${YELLOW}[3/3] Rapid-fire limit test (120 reqs)${NC}"
N=120; i=0; total=0
for ((i=1; i<=120; i++)); do
  hit "GET" "/api/games?club=119" "rapid" &
  if (( i % 20 == 0 )); then wait; fi
done
wait
echo ""
echo ""

# ── summary ──
echo -e "${GREEN}═══ Results ═══${NC}"
echo -e "  Pass: ${GREEN}$pass${NC}  Fail: ${RED}$fail${NC}  Total: $((pass+fail))"
echo ""
echo -e "Now open: ${YELLOW}$BASE/dashboard${NC} to see live metrics."
