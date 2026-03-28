#!/usr/bin/env bash
# demo.sh — logr feature walkthrough for GIF recording
# Run: bash demo.sh
# Tip: record with `vhs demo.tape` or `asciinema rec`

set -euo pipefail

LOGR="LOGR_DEV=1 ./bin/logr"
LOG="testdata/sample.log"

# ── helpers ──────────────────────────────────────────────────────────────────

BOLD="\033[1m"
DIM="\033[2m"
CYAN="\033[36m"
GREEN="\033[32m"
YELLOW="\033[33m"
RESET="\033[0m"

banner() {
  echo
  echo -e "${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  echo -e "${BOLD}  $1${RESET}"
  echo -e "${DIM}  \$ $2${RESET}"
  echo -e "${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  echo
  sleep 1
}

pause() { sleep "${1:-1.5}"; }

# ── clear screen & intro ─────────────────────────────────────────────────────

clear
echo
echo -e "${BOLD}  logr${RESET} — microservice-aware JSON log filter"
echo -e "${DIM}  hierarchical paths · named profiles · noise suppression${RESET}"
echo
pause 2

# ── 1. pretty-print (no filter) ──────────────────────────────────────────────

banner "Pretty-print — raw JSON → human-readable output" \
       "cat service.log | logr"
eval "$LOGR < $LOG"
pause 2

# ── 2. level filter ──────────────────────────────────────────────────────────

banner "Level filter — show only WARN and above" \
       "cat service.log | logr --level warn"
eval "$LOGR --level warn < $LOG"
pause 2

# ── 3. hier path filter ──────────────────────────────────────────────────────

banner "Hier path filter — trace a request through payment.*" \
       "cat service.log | logr --hier \"payment.**\""
eval "$LOGR --hier 'payment.**' < $LOG"
pause 2

# ── 4. service + level combined ──────────────────────────────────────────────

banner "Service + level combined — inventory errors only" \
       "cat service.log | logr --service inventory-service --level error"
eval "$LOGR --service inventory-service --level error < $LOG"
pause 2

# ── 5. TTL noise suppression ─────────────────────────────────────────────────

banner "Noise suppression — deduplicate repeated log patterns" \
       "cat service.log | logr --suppress-ttl 60s"
echo -e "${DIM}  (\"processed 42 items\" and \"processed 99 items\" share the same fingerprint)${RESET}"
echo
eval "$LOGR --suppress-ttl 60s < $LOG"
pause 2

# ── 6. field include ─────────────────────────────────────────────────────────

banner "Field filter — trace a single request across all services" \
       "cat service.log | logr --include \"request_id=req-abc-123\""
eval "$LOGR --include 'request_id=req-abc-123' < $LOG"
pause 2

# ── 7. profile save ──────────────────────────────────────────────────────────

banner "Save a profile — persist filter config for reuse" \
       "logr --service payment-service --level warn --save-profile payment-errors"
eval "$LOGR --service payment-service --level warn --hier 'payment.**' --save-profile payment-errors"
pause 1

echo -e "${DIM}  ~/.logr/profiles/payment-errors.toml${RESET}"
echo
cat ~/.logr/profiles/payment-errors.toml
pause 2

# ── 8. profile load ──────────────────────────────────────────────────────────

banner "Load a profile — one flag instead of many" \
       "cat service.log | logr --profile payment-errors"
eval "$LOGR --profile payment-errors < $LOG"
pause 2

# ── 9. profile list ──────────────────────────────────────────────────────────

banner "List profiles" \
       "logr profile list"
eval "$LOGR profile list"
pause 2

# ── 10. JSON output ──────────────────────────────────────────────────────────

banner "JSON output — pipe into jq for further processing" \
       "cat service.log | logr --level error --json | jq ."
eval "$LOGR --level error --json < $LOG" | head -8 | jq .
pause 2

# ── outro ────────────────────────────────────────────────────────────────────

echo
echo -e "${BOLD}${GREEN}  logr${RESET}${BOLD} — \$19 one-time · single binary · no dependencies${RESET}"
echo -e "${DIM}  linux · macOS · windows · amd64 · arm64${RESET}"
echo
