#!/usr/bin/env bash
# Post-deploy PUBLIC verification for frankfurter-fx (SPEC02 Phase 3) + Phase 4/5 helper.
# Platform-agnostic: works for Render / Fly / any public https host.
#
#   usage: ./verify_public.sh <HOST>
#   e.g.   ./verify_public.sh https://frankfurter-fx.onrender.com
#
# It does NOT spend funds and does NOT make paid calls. It:
#   1. warms the instance (free tiers sleep -> cold start; retry until /healthz is 200)
#   2. asserts GET /healthz -> 200
#   3. asserts unpaid GET /v1/latest?base=USD -> 402, decodes the Payment-Required
#      header and asserts Kite testnet params (network/pieUSD/version/amount)
#   4. prints the verified kpass 1.10.1 commands for the real paid call (Phase 4)
#      and reverse-no-settle (Phase 5), with <HOST> filled in.
set -euo pipefail

HOST="${1:-}"
[ -n "$HOST" ] || { echo "usage: verify_public.sh <https-host>" >&2; exit 2; }
case "$HOST" in https://*) : ;; *) HOST="https://$HOST" ;; esac
HOST="${HOST%/}"

command -v curl >/dev/null 2>&1 || { echo "ERROR: curl not found" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "ERROR: python3 not found" >&2; exit 1; }

say() { printf '\n=== %s ===\n' "$1"; }

say "Phase 3a: warm + verify /healthz -> 200 (free tier may cold-start ~50s)"
HZ_CODE="000"
for i in $(seq 1 18); do
  HZ_CODE="$(curl -s -o /tmp/fx-hz.json -w '%{http_code}' --max-time 20 "$HOST/healthz" || true)"
  [ "$HZ_CODE" = "200" ] && break
  echo "  attempt $i: /healthz -> $HZ_CODE (retrying in 5s)"; sleep 5
done
[ "$HZ_CODE" = "200" ] || { echo "ERROR: /healthz never returned 200 (last=$HZ_CODE). Body: $(cat /tmp/fx-hz.json 2>/dev/null)" >&2; exit 1; }
echo "GET $HOST/healthz -> 200"
cat /tmp/fx-hz.json; echo

say "Phase 3b: verify unpaid /v1/latest -> 402 + decode header"
HDR=/tmp/fx-402.hdr; PR=/tmp/fx-pr.b64
UP_CODE="$(curl -s -o /dev/null -D "$HDR" -w '%{http_code}' --max-time 30 "$HOST/v1/latest?base=USD")"
[ "$UP_CODE" = "402" ] || { echo "ERROR: unpaid /v1/latest returned $UP_CODE (expected 402)" >&2; exit 1; }
echo "GET $HOST/v1/latest?base=USD (unpaid) -> 402"
grep -i '^payment-required:' "$HDR" | head -1 | cut -d: -f2- | tr -d ' \r' > "$PR"
[ -s "$PR" ] || { echo "ERROR: no Payment-Required header captured" >&2; exit 1; }

python3 - "$PR" <<'PY'
import base64, json, sys
raw = open(sys.argv[1]).read().strip()
pad = raw + '=' * (-len(raw) % 4)
try:
    d = json.loads(base64.b64decode(pad))
except Exception:
    d = json.loads(base64.urlsafe_b64decode(pad))
def mask(o):
    if isinstance(o, dict): return {k: mask(v) for k, v in o.items()}
    if isinstance(o, list): return [mask(v) for v in o]
    if isinstance(o, str) and o.startswith('0x') and len(o) >= 12: return o[:6]+'...'+o[-4:]
    return o
print(json.dumps(mask(d), indent=2, ensure_ascii=False))
acc = (d.get('accepts') or [{}])[0]; extra = acc.get('extra') or {}
amount = acc.get('maxAmountRequired', acc.get('amount'))
checks = [
    ('accepts[0].network == eip155:2368', acc.get('network') == 'eip155:2368'),
    ('extra.name == pieUSD', extra.get('name') == 'pieUSD'),
    ('extra.version == 1', str(extra.get('version')) == '1'),
    ('amount == 1000000000000000', str(amount) == '1000000000000000'),
]
for label, ok in checks: print(('PASS' if ok else 'FAIL'), '-', label)
ok_all = all(ok for _, ok in checks)
print('RESULT:', 'ALL_PASS' if ok_all else 'SOME_FAIL')
sys.exit(0 if ok_all else 1)
PY

say "PUBLIC VERIFICATION PASSED"
cat <<EOF
Host: $HOST

NOTE (Render free): the instance sleeps after ~15 min idle. Before EACH kpass paid
call below, re-warm it:  curl -s $HOST/healthz   (wait for 200), then run execute.

Next MANUAL steps (verified kpass 1.10.1 syntax: COLON form, no 'sandbox' command):
  Phase 4  real testnet paid call (free faucet pieUSD; no real money):
    kpass health && kpass me                 # preflight; if not logged in:
    kpass login init --email <YOUR_EMAIL> && kpass login verify --login-id <id> --code <OTP>
    kpass wallet address                     # payer addr -> fund it (testnet, free):
    kpass faucet drop --recipient <PAYER_ADDR> --token pieUSD && kpass wallet balance
    kpass agent:register --type <type>
    kpass agent:session create --max-amount-per-tx <amt> --ttl <dur> --assets pieUSD
    kpass agent:session status --request-id <id> --wait        # requires approval
    kpass agent:session use --session-id <id>
    curl -s $HOST/healthz                    # WARM the free instance first
    kpass agent:session execute --url "$HOST/v1/latest?base=USD"
    -> expect 200 + upstream FX body + settle tx hash; verify payee == PAY_TO on testnet.kitescan.ai
  Phase 5  reverse-no-settle (reuse session):
    curl -s $HOST/healthz                    # WARM again
    kpass agent:session execute --url "$HOST/v1/latest?base=NOTACURRENCY"   # upstream 404
    -> expect 4xx and NO new settle tx (payer balance unchanged)
  Phase 6  service.yaml: status draft -> testnet, base_url: "$HOST"; replace
           maintainer.github placeholder with your real board-bound GitHub owner; npm run validate
  Phase 7  git commit (review diff for secrets first) + push to default branch + board submission
EOF
