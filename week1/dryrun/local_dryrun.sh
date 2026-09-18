#!/usr/bin/env bash
# Reproduce SPEC02 §3-§4 locally: native build+vet, the Dockerfile build stage
# (linux/amd64 cross-compile), then dry-run the service for healthz 200 + unpaid 402
# and decode/assert the PAYMENT-REQUIRED header. Deploys nothing, spends nothing.
# Requires: go (g-managed at ~/.g/go/bin/go), curl, python3.
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
SVC_DIR="${SVC_DIR:-$(cd "$HERE/../frankfurter-fx" && pwd)}"
GO="${GO:-$HOME/.g/go/bin/go}"

cd "$SVC_DIR"
echo "=== go version ==="; "$GO" version
# Build the native binary to /tmp (NOT the source dir) so no stray executable is left behind.
echo "=== native build + vet ==="
"$GO" build -o /tmp/fx-native . && "$GO" vet ./... && echo "build+vet OK"

echo "=== Dockerfile build stage: CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ==="
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "$GO" build -o /tmp/fx-linux .
file /tmp/fx-linux

echo "=== runtime dry-run (testnet .env) ==="
set -a; . ./.env; set +a
P="${PORT:-8080}"
/tmp/fx-native > /tmp/fx-dry.log 2>&1 &
SVC=$!
trap 'kill "$SVC" 2>/dev/null || true' EXIT
for _ in $(seq 1 40); do curl -sf "http://localhost:$P/healthz" >/dev/null 2>&1 && break; sleep 0.5; done

echo "-- GET /healthz (expect 200) --"
curl -s -o /tmp/fx-health.json -w "HTTP %{http_code}\n" "http://localhost:$P/healthz"
cat /tmp/fx-health.json; echo

echo "-- GET /v1/latest?base=USD unpaid (expect 402) --"
curl -s -D /tmp/fx-hdrs.txt -o /dev/null -w "HTTP %{http_code}\n" "http://localhost:$P/v1/latest?base=USD"
grep -i 'payment-required' /tmp/fx-hdrs.txt | head -1 | sed 's/^[^:]*:[[:space:]]*//' | tr -d '\r' > /tmp/fx-pr.b64 || true

echo "-- decode + assert --"
python3 "$HERE/decode_payment_required.py" /tmp/fx-pr.b64
