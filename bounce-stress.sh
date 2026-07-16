#!/bin/bash
# bounce-stress.sh — 2-minute stress test: 200+ simulated users browsing
# Usage: ./bounce-stress.sh [host] [users] [duration_secs]
#   host defaults to localhost:3001
#   users defaults to 200
#   duration defaults to 120 (seconds)

HOST="${1:-localhost:3001}"
USERS="${2:-200}"
DURATION="${3:-120}"
BASE="http://$HOST"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'

echo -e "${GREEN}═══ Bounce Stress Test ═══${NC}"
echo -e "Host: ${CYAN}$BASE${NC}   Users: ${CYAN}$USERS${NC}   Duration: ${CYAN}${DURATION}s${NC}"
echo ""

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

# Stats trackers
reqs_per_sec=()
for ((i=0;i<=DURATION;i++)); do reqs_per_sec[$i]=0; done

# Worker: browse like a real user
browse() {
  local id=$1
  while $running; do
    for ep in "${ENDPOINTS[@]}"; do
      $running || return
      read -r method path <<< "$ep"
      code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "$BASE$path" 2>/dev/null)
      if [ "$code" = "200" ] || [ "$code" = "302" ]; then
        ((ok++))
      else
        ((err++))
      fi
      ((total++))
      local idx=$((SECONDS - start_time))
      [ "$idx" -ge 0 ] 2>/dev/null && [ "$idx" -le "$DURATION" ] 2>/dev/null && ((reqs_per_sec[idx]++))
      # Simulate read time (0.1-0.5s)
      sleep "0.$(( RANDOM % 40 + 10 ))" 2>/dev/null || sleep 0.2 2>/dev/null || true
    done
  done
}

echo -e "${YELLOW}Launching $USERS workers...${NC}"
start_time=$SECONDS

# Spawn workers in batches to avoid fork bombs
BATCH=20
for ((i=0; i<USERS; i+=BATCH)); do
  end=$((i + BATCH))
  [ "$end" -gt "$USERS" ] && end=$USERS
  for ((j=i; j<end; j++)); do
    browse "$j" &
  done
  sleep 0.3 2>/dev/null || true
done

# Display loop
while [ $SECONDS -lt $deadline ]; do
  elapsed=$((SECONDS - start_time))
  remaining=$((DURATION - elapsed))
  rps=${reqs_per_sec[$elapsed]:-0}
  total_rps=$(( total / (elapsed + 1) ))
  printf "\r${CYAN}%s${NC}  %4ds | total=%6d ok=${GREEN}%6d${NC} err=${RED}%4d${NC} | rps=%4d avg=%4d | %3ds  " \
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
