#!/bin/bash
# bounce-stress.sh — 2-minute stress test: 100+ simulated users browsing
# Usage: ./bounce-stress.sh [host] [users] [duration_secs]
#   host defaults to localhost:3001
#   users defaults to 120
#   duration defaults to 120 (seconds)

HOST="${1:-localhost:3001}"
USERS="${2:-120}"
DURATION="${3:-120}"
BASE="http://$HOST"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'

echo -e "${GREEN}═══ Bounce Stress Test ═══${NC}"
echo -e "Host: ${CYAN}$BASE${NC}   Users: ${CYAN}$USERS${NC}   Duration: ${CYAN}${DURATION}s${NC}"
echo ""

# Browse-like endpoints (simulating real user navigation)
ENDPOINTS=(
  "GET /dashboard"
  "GET /docs"
  "GET /test"
  "GET /health"
  "GET /metrics"
  "GET /api/competitions"
  "GET /api/games?club=119&season=2025/2026"
  "GET /api/games?club=127"
  "GET /api/games/today"
  "GET /api/games/live"
  "GET /api/standings/10902"
  "GET /api/elo"
  "GET /api/predictions/413420"
  "GET /api/h2h?team_a=127&team_b=120"
  "GET /api/athlete/269564"
  "GET /api/team/equipa_57682"
  "GET /api/club/127/teams"
  "GET /api/tugabasket/standings?competitionId=1"
  "GET /api/tugabasket/players?competitionId=1"
  "GET /api/tugabasket/teams?competitionId=1"
)

spinner=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)
total=0 ok=0 err=0
running=true
deadline=$((SECONDS + DURATION))

# Stats trackers per second
declare -a reqs_per_sec
for ((i=0;i<DURATION;i++)); do reqs_per_sec[$i]=0; done
sec_total=0

# Worker: browse like a real user
browse() {
  local id=$1
  while $running; do
    for ep in "${ENDPOINTS[@]}"; do
      $running || break
      read method path <<< "$ep"
      code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "$BASE$path" 2>/dev/null)
      if [ "$code" = "200" ] || [ "$code" = "302" ]; then
        ((ok++))
      else
        ((err++))
      fi
      ((total++))
      ((reqs_per_sec[$((SECONDS - start_time))]++))
      # Simulate read time (100-500ms)
      sleep 0.$(( RANDOM % 40 + 10 )) 2>/dev/null || true
    done
  done
}

echo -e "${YELLOW}Launching $USERS workers...${NC}"
start_time=$SECONDS

# Spawn workers
for ((i=0; i<USERS; i++)); do
  browse "$i" &
done

# Display loop
while [ $SECONDS -lt $deadline ]; do
  elapsed=$((SECONDS - start_time))
  remaining=$((DURATION - elapsed))
  rps=${reqs_per_sec[$elapsed]:-0}
  total_rps=$((total / (elapsed + 1)))
  printf "\r${CYAN}%s${NC}  %4ds | total=%5d ok=${GREEN}%5d${NC} err=${RED}%3d${NC} | rps=%4d avg=%4d | remaining=%3ds  " \
    "${spinner[$((elapsed%10))]}" "$elapsed" "$total" "$ok" "$err" "$rps" "$total_rps" "$remaining"
done

running=false
wait 2>/dev/null

elapsed=$((SECONDS - start_time))
echo ""
echo ""
echo -e "${GREEN}═══ Done ═══${NC}"
echo -e "  Duration: ${elapsed}s"
echo -e "  Total requests: ${CYAN}$total${NC}"
echo -e "  OK:             ${GREEN}$ok${NC}"
echo -e "  Errors:         ${RED}$err${NC}"
echo -e "  Avg req/s:      $(( total / (elapsed + 1) ))"
echo ""
echo -e "Dashboard: ${YELLOW}$BASE/dashboard${NC}"
