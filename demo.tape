Output demo.gif

Set Shell "bash"
Set FontSize 13
Set Width 1100
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 40ms
Set PlaybackSpeed 0.8

# Start in the right directory
Env LOGR_DEV "1"
Env PATH "/opt/homebrew/bin:/usr/bin:/bin"

Hide
Type "cd /Users/saurabh.saini/Desktop/projects/weekend/logr"
Enter
Sleep 500ms
Show

# ── Intro ─────────────────────────────────────────────────────────────────
Sleep 1s

Type "# logr — microservice-aware JSON log filter"
Sleep 1.5s
Enter

Sleep 800ms

# ── Scene 1: Pretty-print ─────────────────────────────────────────────────
Type "# 1. Pretty-print raw JSON logs"
Enter
Sleep 500ms
Type "cat testdata/sample.log | ./bin/logr"
Sleep 800ms
Enter
Sleep 3s

# ── Scene 2: Level filter ─────────────────────────────────────────────────
Type "# 2. Show only WARN and above"
Enter
Sleep 500ms
Type "cat testdata/sample.log | ./bin/logr --level warn"
Sleep 800ms
Enter
Sleep 2.5s

# ── Scene 3: Hier path ────────────────────────────────────────────────────
Type "# 3. Trace all payment.** path entries"
Enter
Sleep 500ms
Type `cat testdata/sample.log | ./bin/logr --hier "payment.**"`
Sleep 800ms
Enter
Sleep 2.5s

# ── Scene 4: Service + level ──────────────────────────────────────────────
Type "# 4. inventory-service errors only"
Enter
Sleep 500ms
Type "cat testdata/sample.log | ./bin/logr --service inventory-service --level error"
Sleep 800ms
Enter
Sleep 2.5s

# ── Scene 5: TTL suppression ──────────────────────────────────────────────
Type "# 5. Deduplicate repeated log patterns (TTL suppression)"
Enter
Sleep 500ms
Type "cat testdata/sample.log | ./bin/logr --suppress-ttl 60s"
Sleep 800ms
Enter
Sleep 3s

# ── Scene 6: Field include ────────────────────────────────────────────────
Type "# 6. Trace a single request across all services"
Enter
Sleep 500ms
Type `cat testdata/sample.log | ./bin/logr --include "request_id=req-abc-123"`
Sleep 800ms
Enter
Sleep 2.5s

# ── Scene 7: Save profile ─────────────────────────────────────────────────
Type "# 7. Save filter config as a named profile"
Enter
Sleep 500ms
Type `./bin/logr --service payment-service --level warn --hier "payment.**" --save-profile payment-errors`
Sleep 800ms
Enter
Sleep 1.5s
Type "cat ~/.logr/profiles/payment-errors.toml"
Sleep 600ms
Enter
Sleep 2s

# ── Scene 8: Load profile ─────────────────────────────────────────────────
Type "# 8. Reuse it — one flag instead of many"
Enter
Sleep 500ms
Type "cat testdata/sample.log | ./bin/logr --profile payment-errors"
Sleep 800ms
Enter
Sleep 2.5s

# ── Scene 9: JSON output ──────────────────────────────────────────────────
Type "# 9. JSON output — pipe into jq"
Enter
Sleep 500ms
Type "cat testdata/sample.log | ./bin/logr --level error --json | jq ."
Sleep 800ms
Enter
Sleep 3s

# ── Outro ─────────────────────────────────────────────────────────────────
Type "# logr — $19 one-time · single binary · no deps"
Enter
Sleep 3s
