#!/bin/bash
# bounce-test-all.sh — Test all Bounce endpoints, save responses for review
# Usage: ./bounce-test-all.sh [host]
#   host defaults to localhost:3001

HOST="${1:-localhost:3001}"
BASE="http://$HOST"
OUT="bounce-test-results"
rm -rf "$OUT"
mkdir -p "$OUT"
# Wait for server to be ready
for i in 1 2 3 4 5; do curl -s "$BASE/health" > /dev/null 2>&1 && break; sleep 1; done

GREEN='\033[0;32m'; RED='\033[0;31m'; CYAN='\033[0;36m'; NC='\033[0m'
pass=0; fail=0

test_ep() {
  local method="$1" path="$2" label="$3" body="$4"
  local file="$OUT/${label}.json"
  local code
  if [ "$method" = "POST" ]; then
    code=$(curl -s -o "$file" -w '%{http_code}' --max-time 15 -X POST -H "Content-Type: application/json" -d "$body" "$BASE$path" 2>/dev/null)
  else
    code=$(curl -s -o "$file" -w '%{http_code}' --max-time 15 "$BASE$path" 2>/dev/null)
  fi
  if [ "$code" = "200" ] || [ "$code" = "302" ]; then
    printf "  ${GREEN}✓${NC} %-50s ${CYAN}%s${NC}\n" "$label" "$code"
    ((pass++))
  else
    printf "  ${RED}✗${NC} %-50s ${RED}%s${NC}\n" "$label" "$code"
    ((fail++))
  fi
}

echo "══════════════════════════════════════════════"
echo "  Bounce API Test Suite — $BASE"
echo "  Results saved to: $OUT/"
echo "══════════════════════════════════════════════"
echo ""

# Wait for server to be ready
sleep 2
echo "── System ──"
test_ep GET "/health" "health"
test_ep GET "/metrics" "metrics"
echo ""

echo "── Competitions ──"
test_ep GET "/api/competitions" "competitions"
test_ep GET "/api/competition/10902/mvp" "comp-mvp"
echo ""

echo "── Games ──"
test_ep GET "/api/games?club=119&season=2025/2026" "games-benfica"
test_ep GET "/api/games?club=127" "games-fcp"
test_ep GET "/api/games/today" "games-today"
test_ep GET "/api/games/live" "games-live"
test_ep GET "/api/game/413420" "game-detail"
echo ""

echo "── Standings ──"
test_ep GET "/api/standings/10902" "standings"
echo ""

echo "── Athletes ──"
test_ep GET "/api/athlete/269564" "athlete"
echo ""

echo "── Teams & Clubs ──"
test_ep GET "/api/team/equipa_57682" "team"
test_ep GET "/api/club/127/teams" "club-teams"
echo ""

echo "── ELO & Insights ──"
test_ep GET "/api/elo" "elo"
test_ep GET "/api/predictions/413420" "predictions"
test_ep GET "/api/h2h?team_a=127&team_b=120" "h2h"
echo ""

echo "── TugaBasket ──"
test_ep GET "/api/tugabasket/standings?competitionId=1" "tb-standings"
test_ep GET "/api/tugabasket/players?competitionId=1" "tb-players"
test_ep GET "/api/tugabasket/teams?competitionId=1" "tb-teams"
echo ""

echo "── Metrics History ──"
test_ep GET "/api/metrics/history?metric=requests&since=1h" "metrics-history"
test_ep GET "/api/metrics/history/simple?minutes=5" "metrics-simple"
echo ""

echo "── Pages ──"
test_ep GET "/dashboard" "page-dashboard"
test_ep GET "/docs" "page-docs"
test_ep GET "/test" "page-test"
test_ep GET "/" "page-root"
echo ""

echo "── Batch ──"
test_ep POST "/api/batch" "batch" '[{"path":"/api/games?club=119"},{"path":"/api/standings/10902"},{"path":"/api/elo"}]'
echo ""

echo "══════════════════════════════════════════════"
echo "  Pass: ${GREEN}$pass${NC}  Fail: ${RED}$fail${NC}  Total: $((pass+fail))"
echo "  Results: $OUT/"
echo "══════════════════════════════════════════════"
