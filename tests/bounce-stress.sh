#!/bin/bash
# bounce-stress.sh — Aggressive stress test for Bounce
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
  "/dashboard"
  "/docs"
  "/test"
  "/health"
  "/metrics"
  "/api/competitions"
  "/api/games?club=119&season=2025/2026"
  "/api/games?club=127"
  "/api/games/today"
  "/api/games/live"
  "/api/standings/10902"
  "/api/elo"
  "/api/predictions/413420"
  "/api/h2h?team_a=127&team_b=120"
  "/api/athlete/269564"
  "/api/team/equipa_57682"
  "/api/club/127/teams"
  "/api/tugabasket/standings?competitionId=1"
  "/api/tugabasket/players?competitionId=1"
  "/api/tugabasket/teams?competitionId=1"
)

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT
running=true

spinner=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)

# Worker: writes results to a temp file
browse() {
  local id=$1
  local f="$TMPDIR/w$id"
  local o=0 e=0 t=0
  local deadline=$((SECONDS + DURATION))
  while [ $SECONDS -lt $deadline ]; do
    for ep in "${ENDPOINTS[@]}"; do
      [ $SECONDS -ge $deadline ] && break
      code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "$BASE$ep" 2>/dev/null)
      if [ "$code" = "200" ] || [ "$code" = "302" ]; then ((o++)); else ((e++)); fi
      ((t++))
    done
    sleep 0.$(( RANDOM % 30 + 5 )) 2>/dev/null || sleep 0.1 2>/dev/null || true
  done
  echo "$t $o $e" > "$f"
}

echo -e "${YELLOW}Launching $USERS workers...${NC}"
start_time=$SECONDS

for ((i=0; i<USERS; i+=20)); do
  end=$((i + 20))
  [ $end -gt $USERS ] && end=$USERS
  for ((j=i; j<end; j++)); do
    browse "$j" &
  done
  sleep 0.2 2>/dev/null || true
  printf "\r  Launched %d/%d" $end $USERS
done
echo ""

# Display loop
while [ $SECONDS -lt $((start_time + DURATION)) ]; do
  elapsed=$((SECONDS - start_time))
  remaining=$((DURATION - elapsed))
  running_count=$(jobs -r | wc -l)
  # Quick estimate from temp files
  total=0 ok=0
  for f in "$TMPDIR"/w*; do
    [ -f "$f" ] || continue
    read t o e < "$f" 2>/dev/null
    total=$((total + t))
    ok=$((ok + o))
  done
  rps=$(( total / (elapsed + 1) ))
  printf "\r  ${spinner[$((elapsed%10))]}  %4ds | running=%3d | total=%6d ok=%6d | rps=%4d | %3ds  " \
    "$elapsed" "$running_count" "$total" "$ok" "$rps" "$remaining"
  sleep 0.5
done

running=false
wait 2>/dev/null

# Final count
total=0 ok=0 err=0
for f in "$TMPDIR"/w*; do
  [ -f "$f" ] || continue
  read t o e < "$f" 2>/dev/null
  total=$((total + t))
  ok=$((ok + o))
  err=$((err + e))
done

elapsed=$((SECONDS - start_time))
echo ""
echo ""
echo -e "${GREEN}═══ Done ═══${NC}"
echo -e "  Duration:       ${elapsed}s"
echo -e "  Total requests: ${CYAN}$total${NC}"
echo -e "  OK:             ${GREEN}$ok${NC}"
echo -e "  Errors:         ${RED}$err${NC}"
[ $elapsed -gt 0 ] && echo -e "  Avg req/s:      $(( total / elapsed ))"
echo ""
echo -e "Dashboard: ${YELLOW}$BASE/dashboard${NC}"
