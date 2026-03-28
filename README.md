# logr

A fast, microservice-aware JSON log filter with saved profiles.

```
kubectl logs -f my-pod | logr --level warn
logr service.log --hier "payment.**" --tui
logr service.log --follow --service payment-service
```

---

## Why logr

Every existing tool (`hl`, `humanlog`, `klp`, `lnav`) pretty-prints JSON logs. None of them have:

| Feature | logr | others |
|---|---|---|
| Dot-path hierarchy filter (`payment.txn.*`) | ✅ | ❌ |
| Named profiles (save/load filter configs) | ✅ | ❌ |
| TTL noise suppression (dedup repeated patterns) | ✅ | ❌ |
| Interactive TUI with live search | ✅ | ❌ |
| File follow mode (`tail -f` style) | ✅ | ❌ |
| Key projection (show only selected fields) | ✅ | ❌ |

---

## Installation

Download the binary for your platform from the releases page and put it on your `PATH`:

```bash
# macOS (Apple Silicon)
tar xz logr_darwin_arm64.tar.gz
sudo mv logr /usr/local/bin/

# macOS (Intel)
tar xz logr_darwin_amd64.tar.gz
sudo mv logr /usr/local/bin/

# Linux (amd64)
tar xz logr_linux_amd64.tar.gz
sudo mv logr /usr/local/bin/
```

### License

On first run, activate your Gumroad license key:

```bash
logr --license XXXX-XXXX-XXXX-XXXX
# License verified and cached for 30 days.
```

The key is cached in `~/.logr/license` for 30 days. You only need to re-run this after the cache expires or on a new machine.

---

## Quick start

logr reads from **stdin** or a **file argument**:

```bash
# From stdin
kubectl logs -f my-pod | logr --level warn
cat service.log | logr --service payment-service

# From a file
logr service.log --level error
logr service.log --tui
```

---

## Features

### 1. Pretty-print

Converts raw JSON log lines into a readable, colorized format:

```
[10:00:04.012] INFO  api-gateway  (payment.txn.charge)  incoming request  method=POST request_id=req-abc-123
[10:00:04.800] WARN  payment-service  stripe response latency high  latency_ms=550 threshold_ms=500
[10:00:07.000] FATAL inventory-service  all retries exhausted, circuit breaker opened
```

Fields recognized automatically across common logging libraries:

| Concept | Field aliases recognized |
|---|---|
| Timestamp | `timestamp`, `ts`, `time`, `@timestamp` |
| Level | `level`, `lvl`, `severity` |
| Message | `msg`, `message`, `event` |
| Service | `service`, `svc`, `app`, `component` |
| Hier path | `path`, `hier_path`, `trace` |

Non-JSON lines are passed through as-is (dimmed).

---

### 2. Level filter (`--level` / `-l`)

Show only entries at or above a minimum severity:

```bash
cat service.log | logr --level warn      # WARN, ERROR, FATAL
cat service.log | logr --level error     # ERROR, FATAL only
cat service.log | logr -l info
```

Levels in order: `debug` < `info` < `warn` < `error` < `fatal`

---

### 3. Service filter (`--service` / `-s`)

Show only logs from specific services:

```bash
cat service.log | logr --service payment-service
cat service.log | logr --service payment-service,auth-service
cat service.log | logr -s api-gateway -s inventory-service
```

---

### 4. Field filters (`--include` / `--exclude`)

**Include** — show only entries where a field matches a value:

```bash
cat service.log | logr --include "request_id=req-abc-123"  # exact match
cat service.log | logr --include "upstream=*"              # field exists
cat service.log | logr -i "status=500" -i "status=503"    # either match
```

**Exclude** — drop entries where a field matches:

```bash
cat service.log | logr --exclude "level=debug"
cat service.log | logr -e "service=health-check"
```

---

### 5. Hierarchical path filter (`--hier` / `-H`)

Filter by dot-path `hier_path` field using `*` (one segment) and `**` (any depth) wildcards:

```bash
# All payment paths at any depth
cat service.log | logr --hier "payment.**"

# Only direct children of payment.txn
cat service.log | logr --hier "payment.txn.*"

# Multiple patterns (any match passes)
cat service.log | logr -H "payment.**" -H "inventory.**"
```

| Pattern | Matches | Does not match |
|---|---|---|
| `payment.*` | `payment.charge` | `payment.charge.retry` |
| `payment.**` | `payment.charge`, `payment.charge.retry` | `auth.session` |
| `**.check` | `inventory.stock.check`, `health.check` | `inventory.stock` |

---

### 6. TTL noise suppression (`--suppress-ttl`)

Deduplicates repeated log patterns within a time window. Useful for silencing chatty polling loops.

```bash
cat service.log | logr --suppress-ttl 30s
cat service.log | logr --suppress-ttl 1m
```

logr normalizes numbers in messages before fingerprinting:
`"processed 42 items"` and `"processed 99 items"` are treated as the same pattern and deduplicated.

---

### 7. Key projection (`--keys` / `-k`)

Show only specific extra fields per log line. Core fields (timestamp, level, service, message) are always shown.

```bash
# Only show request_id and latency_ms, drop all other fields
cat service.log | logr --keys request_id,latency_ms

# Works with --json too
cat service.log | logr --keys timestamp,level,msg,request_id --json
```

---

### 8. File input

Pass a file path directly instead of piping:

```bash
logr service.log
logr service.log --level error --service payment-service
logr service.log --hier "payment.**" --keys request_id,amount
```

---

### 9. Follow mode (`--follow` / `-f`)

Watch a file for new log lines as they are written — like `tail -f` but with all logr filters applied. Handles log rotation automatically.

```bash
logr service.log --follow
logr service.log --follow --level error
logr service.log --follow --service payment-service --suppress-ttl 30s
```

> Requires a file argument. Does not work with stdin.

---

### 10. Interactive TUI (`--tui` / `-T`)

Opens a full-screen interactive viewer for a log file or stdin:

```bash
logr service.log --tui
logr service.log --level warn --tui
cat service.log | logr --tui
```

**Keyboard shortcuts:**

| Key | Action |
|---|---|
| `↑` / `k` | Scroll up |
| `↓` / `j` | Scroll down |
| `PgUp` / `PgDn` | Page up / down |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `/` | Open live search (filters as you type) |
| `Enter` | Apply search |
| `Esc` | Clear search |
| `q` / `Ctrl+C` | Quit |

The header shows `matched / total` entry counts. The search bar filters across the entire raw log line (all fields).

---

### 11. Profiles

Save any combination of flags as a named profile and reuse it later.

**Save:**

```bash
# Save current flags as a profile
logr --service payment-service --level warn --hier "payment.**" --save-profile payment-errors

# Save with suppression and key projection
logr -s inventory-service -l error --suppress-ttl 1m --keys request_id,upstream --save-profile inv-errors
```

**Load:**

```bash
cat service.log | logr --profile payment-errors
logr service.log --profile inv-errors --tui
```

CLI flags always override the loaded profile:

```bash
# Profile has level=warn, override to debug for this run
cat service.log | logr --profile payment-errors --level debug
```

**Manage profiles:**

```bash
logr profile list            # list all saved profiles
logr profile load <name>     # print a profile's contents
logr profile delete <name>   # delete a profile
```

Profiles are stored as TOML files in `~/.logr/profiles/`.

---

### 12. JSON output (`--json`)

Emit each matching entry as a JSON object instead of pretty-printing. Useful for piping into `jq` or other tools.

```bash
cat service.log | logr --level error --json
cat service.log | logr --level error --json | jq '.message'
cat service.log | logr --keys timestamp,level,message,request_id --json
```

---

## Combining flags

All flags compose freely:

```bash
# Trace a specific request across all services, errors only, in TUI
logr service.log \
  --include "request_id=req-abc-123" \
  --level error \
  --tui

# Follow payment service live, suppress noise, project key fields
logr service.log \
  --follow \
  --service payment-service \
  --suppress-ttl 30s \
  --keys request_id,amount,charge_id

# Save a complex profile, then use it
logr \
  --service payment-service \
  --level warn \
  --hier "payment.**" \
  --suppress-ttl 1m \
  --keys request_id,amount \
  --save-profile payment-warn

cat service.log | logr --profile payment-warn
```

---

## Output modes

| Mode | Flag | Use case |
|---|---|---|
| Pretty-print (default) | — | Human reading in terminal |
| No color | `--no-color` | Piping to files, CI logs |
| JSON | `--json` | Piping to `jq`, further processing |
| TUI | `--tui` | Interactive exploration |

---

## Configuration

logr reads optional defaults from `~/.logr/config.toml`:

```toml
license_key = "XXXX-XXXX-XXXX-XXXX"
```

---

## Supported log formats

logr works with any JSON logging library out of the box:

| Library | Language | Notes |
|---|---|---|
| `zap` | Go | `ts`, `level`, `msg` |
| `zerolog` | Go | `time`, `level`, `message` |
| `logrus` | Go | `time`, `level`, `msg` |
| `pino` | Node.js | `time`, `level`, `msg` |
| `winston` | Node.js | `timestamp`, `level`, `message` |
| `structlog` | Python | `timestamp`, `level`, `event` |
| Spring Boot | Java | `@timestamp`, `level`, `message` |

---

## License

Copyright (c) 2026 Saurabh Saini. All rights reserved.

Source code is provided for reference and audit purposes only. Commercial use requires a valid license purchased at [bearking11.gumroad.com](https://bearking11.gumroad.com/). See [LICENSE](./LICENSE) for full terms.

---

## Build from source

Requires Go 1.22+.

```bash
git clone <repo-url>
cd logr
go build -o logr .
```

Cross-platform releases via [GoReleaser](https://goreleaser.com/):

```bash
goreleaser release --snapshot --clean
```

Binaries are produced for: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`.
