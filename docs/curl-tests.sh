#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_URL="$BASE_URL/api/v1"
OUT_DIR="${OUT_DIR:-/tmp/casemind-curl-test}"

EMAIL="${EMAIL:-detective.$(date +%s)@example.com}"
PASSWORD="${PASSWORD:-Password123!}"
DISPLAY_NAME="${DISPLAY_NAME:-Test Detective}"

mkdir -p "$OUT_DIR"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing dependency: $1"
    exit 1
  }
}

need curl
need jq

step() {
  printf "\n==> %s\n" "$1"
}

api() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local outfile="$4"

  if [[ -n "$body" ]]; then
    curl -sS -X "$method" "$API_URL$path" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer ${ACCESS_TOKEN:-}" \
      -d "$body" | tee "$outfile" | jq .
  else
    curl -sS -X "$method" "$API_URL$path" \
      -H "Authorization: Bearer ${ACCESS_TOKEN:-}" \
      | tee "$outfile" | jq .
  fi
}

api_public() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local outfile="$4"

  if [[ -n "$body" ]]; then
    curl -sS -X "$method" "$API_URL$path" \
      -H "Content-Type: application/json" \
      -d "$body" | tee "$outfile" | jq .
  else
    curl -sS -X "$method" "$API_URL$path" \
      | tee "$outfile" | jq .
  fi
}

step "Health checks"
curl -sS "$BASE_URL/health" | tee "$OUT_DIR/health.json" | jq .
curl -sS "$BASE_URL/ready" | tee "$OUT_DIR/ready.json" | jq .

step "Auth: register"
api_public POST /auth/register "{
  \"email\": \"$EMAIL\",
  \"password\": \"$PASSWORD\",
  \"display_name\": \"$DISPLAY_NAME\"
}" "$OUT_DIR/register.json"

ACCESS_TOKEN="$(jq -r '.data.tokens.access_token' "$OUT_DIR/register.json")"
REFRESH_TOKEN="$(jq -r '.data.tokens.refresh_token' "$OUT_DIR/register.json")"

step "Auth: login"
api_public POST /auth/login "{
  \"email\": \"$EMAIL\",
  \"password\": \"$PASSWORD\"
}" "$OUT_DIR/login.json"

step "Auth: refresh"
api_public POST /auth/refresh "{
  \"refresh_token\": \"$REFRESH_TOKEN\"
}" "$OUT_DIR/refresh.json"

step "Current user"
api GET /me "" "$OUT_DIR/me.json"

step "Detective profile/history"
api GET /detective/profile "" "$OUT_DIR/detective-profile.json"
api GET /detective/history "" "$OUT_DIR/detective-history.json"

step "Cases: create"
api POST /cases '{
  "type": "murder",
  "difficulty": "easy"
}' "$OUT_DIR/create-case.json"

CASE_ID="$(jq -r '.data.case.id' "$OUT_DIR/create-case.json")"
echo "CASE_ID=$CASE_ID"

step "Cases: wait until open"
STATUS=""
for i in $(seq 1 30); do
  api GET "/cases/$CASE_ID" "" "$OUT_DIR/case-dashboard.json" >/dev/null
  STATUS="$(jq -r '.data.case.status // empty' "$OUT_DIR/case-dashboard.json")"
  echo "case status: ${STATUS:-unknown}"
  [[ "$STATUS" == "open" ]] && break
  sleep 2
done

if [[ "$STATUS" != "open" ]]; then
  echo "Case did not become open in time. Last dashboard response:"
  cat "$OUT_DIR/case-dashboard.json"
  exit 1
fi

step "Cases: list/detail/events/SSE"
api GET /cases "" "$OUT_DIR/cases.json"
api GET "/cases/$CASE_ID" "" "$OUT_DIR/case-dashboard.json"
api GET "/cases/$CASE_ID/events?limit=20" "" "$OUT_DIR/case-events.json"
curl -sS --max-time 3 "$API_URL/cases/$CASE_ID/stream" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  | tee "$OUT_DIR/case-stream.txt" || true

step "Suspects"
api GET "/cases/$CASE_ID/suspects" "" "$OUT_DIR/suspects.json"
SUSPECT_ID="$(jq -r '.data.suspects[0].id // empty' "$OUT_DIR/suspects.json")"
echo "SUSPECT_ID=$SUSPECT_ID"

if [[ -n "$SUSPECT_ID" ]]; then
  api GET "/cases/$CASE_ID/suspects/$SUSPECT_ID" "" "$OUT_DIR/suspect-detail.json"
  api POST "/cases/$CASE_ID/suspects/$SUSPECT_ID/interrogate" '{
    "message": "Where were you when the victim was last seen?"
  }' "$OUT_DIR/interrogate.json"
fi

step "Evidence"
api GET "/cases/$CASE_ID/evidence" "" "$OUT_DIR/evidence-list.json"
EVIDENCE_ID="$(jq -r '.data.evidence[0].id // empty' "$OUT_DIR/evidence-list.json")"
echo "EVIDENCE_ID=$EVIDENCE_ID"

if [[ -n "$EVIDENCE_ID" ]]; then
  api GET "/cases/$CASE_ID/evidence/$EVIDENCE_ID" "" "$OUT_DIR/evidence-detail.json"
  api POST "/cases/$CASE_ID/evidence/$EVIDENCE_ID/inspect" '{
    "question": "What is the most important detail in this evidence?"
  }' "$OUT_DIR/evidence-inspect.json"
fi

step "Timeline"
api GET "/cases/$CASE_ID/timeline" "" "$OUT_DIR/timeline.json"
api POST "/cases/$CASE_ID/timeline/player" '{
  "occurred_at": "2026-01-01T10:00:00Z",
  "title": "Player test event",
  "description": "A test timeline event created by the curl smoke test.",
  "linked_evidence_ids": []
}' "$OUT_DIR/player-event-create.json"

EVENT_ID="$(jq -r '.data.event.id // empty' "$OUT_DIR/player-event-create.json")"
echo "EVENT_ID=$EVENT_ID"

if [[ -n "$EVENT_ID" ]]; then
  api PUT "/cases/$CASE_ID/timeline/player/$EVENT_ID" '{
    "occurred_at": "2026-01-01T11:00:00Z",
    "title": "Updated player test event",
    "description": "Updated by the curl smoke test.",
    "linked_evidence_ids": []
  }' "$OUT_DIR/player-event-update.json"
fi

step "Map"
api GET "/cases/$CASE_ID/map" "" "$OUT_DIR/map.json"

step "Notes"
api GET "/cases/$CASE_ID/notes" "" "$OUT_DIR/notes-list.json"
api POST "/cases/$CASE_ID/notes" '{
  "title": "Curl smoke note",
  "content": "This note was created from docs/curl-tests.sh."
}' "$OUT_DIR/note-create.json"

NOTE_ID="$(jq -r '.data.note.id // empty' "$OUT_DIR/note-create.json")"
echo "NOTE_ID=$NOTE_ID"

if [[ -n "$NOTE_ID" ]]; then
  api PUT "/cases/$CASE_ID/notes/$NOTE_ID" '{
    "title": "Updated curl smoke note",
    "content": "Updated from docs/curl-tests.sh."
  }' "$OUT_DIR/note-update.json"
fi

step "Solve"
if [[ -n "$SUSPECT_ID" ]]; then
  api POST "/cases/$CASE_ID/solve" "{
    \"accused_suspect_id\": \"$SUSPECT_ID\",
    \"motive\": \"Testing the solve endpoint with the first suspect.\",
    \"reasoning\": \"This is a smoke-test accusation, not a real deduction.\"
  }" "$OUT_DIR/solve.json"
fi
api GET "/cases/$CASE_ID/solve-attempts" "" "$OUT_DIR/solve-attempts.json"

step "Delete created player event and note"
if [[ -n "${EVENT_ID:-}" ]]; then
  curl -sS -X DELETE "$API_URL/cases/$CASE_ID/timeline/player/$EVENT_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -o "$OUT_DIR/player-event-delete.txt" \
    -w "DELETE player event HTTP %{http_code}\n"
fi

if [[ -n "${NOTE_ID:-}" ]]; then
  curl -sS -X DELETE "$API_URL/cases/$CASE_ID/notes/$NOTE_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -o "$OUT_DIR/note-delete.txt" \
    -w "DELETE note HTTP %{http_code}\n"
fi

step "Archive case"
api POST "/cases/$CASE_ID/archive" '{}' "$OUT_DIR/archive-case.json"

step "Auth: logout"
api_public POST /auth/logout "{
  \"refresh_token\": \"$REFRESH_TOKEN\"
}" "$OUT_DIR/logout.json"

step "Done"
echo "Responses saved in $OUT_DIR"
