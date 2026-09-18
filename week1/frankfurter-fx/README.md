# frankfurter-fx

European Central Bank reference exchange rates from the free
[Frankfurter](https://www.frankfurter.app/) API, wrapped as an x402 paid service
on the Kite network. Built from the [`go-gin`](../../templates/go-gin) template.

- **Endpoint:** `GET /v1/latest` → proxies `https://api.frankfurter.dev/v1/latest`
- **Price:** `$0.001` per call, charged in the network stablecoin
  (pieUSD on testnet `eip155:2368`, USDC.e on mainnet `eip155:2366`)
- **Upstream auth:** none (Frankfurter is keyless)
- **Settlement:** `verify → upstream → settle`; the charge is settled only when
  Frankfurter answers with a status `< 400`. An unknown `base`/`symbols` code
  returns 404 upstream, so the caller is **not** charged.

## Why the Go template

Frankfurter's real endpoints live under a `/v1` prefix. The Go template's
`httputil.NewSingleHostReverseProxy` joins the `UPSTREAM_URL` base path with the
request path, so `UPSTREAM_URL=https://api.frankfurter.dev/v1` + `/v1/latest`
correctly reaches the upstream `/v1/latest`. (The TypeScript template drops the
base path and would 404 here.)

## Run locally

```bash
cp .env.example .env        # set PAY_TO to your Kite wallet; KITE_NETWORK=testnet
go build -o frankfurter-fx ./...
set -a && source .env && set +a
./frankfurter-fx            # listens on :8080
```

Health and the unpaid challenge:

```bash
curl -i http://localhost:8080/healthz                 # 200, shows network/asset/price
curl -i "http://localhost:8080/v1/latest?base=USD"    # 402 + Payment-Required header
```

## Pay for a call (Kite testnet, sandbox)

```bash
kpass sandbox on
kpass agent:register --type claude
kpass faucet drop --recipient <agent-wallet> --token pieUSD
kpass agent:session create --task-summary "fx test" \
  --max-amount-per-tx 0.05 --max-total-amount 1 --assets pieUSD --ttl 1h
# approve the printed approval_url in a browser, then:
kpass agent:session status --request-id <id> --wait
kpass agent:session use --session-id <session-id>
kpass agent:session execute --method GET --url "http://localhost:8080/v1/latest?base=USD"
```

A successful call returns HTTP 200, the Frankfurter JSON body, and a settlement
transaction hash; the pieUSD lands in `PAY_TO`. Verify the hash on
<https://testnet.kitescan.ai>.

## Deploy

Deployment must be public **https** (Kite Passport fetches server-side; localhost,
plain http, self-signed certs and browser-verification tunnels are rejected). Set
`status: testnet` and `base_url` in `service.yaml` once deployed.

Live deployment: <https://frankfurter-fx.onrender.com> (Render free tier, Docker
runtime, health check `/healthz`; the free instance sleeps after ~15 min idle —
warm it with `curl <host>/healthz` before the first paid call).

## Verified end-to-end (Kite testnet, 2026-09-18)

| Check | Result |
|---|---|
| `GET /healthz` | 200 |
| Unpaid `GET /v1/latest?base=USD` | 402 + `PAYMENT-REQUIRED` (network `eip155:2368`, pieUSD, v1, amount `1000000000000000`) |
| Paid `GET /v1/latest?base=USD` | 200 + Frankfurter body; settle tx [`0x186f8e20…9336f`](https://testnet.kitescan.ai/tx/0x186f8e2090d80f39b0b83883bf3351e0085368fd6a1ad4c08ca2a5f554b9336f), payee == `PAY_TO`, 0.001 pieUSD |
| Paid `GET /v1/latest?base=NOTACURRENCY` (upstream 404) | 404 `{"message":"not found"}`, **no settlement**, payer balance unchanged |

## Known environment issues (observed on Kite testnet)

- **Testnet head lag breaks the default signing window.** The chain was producing
  blocks ~36 min apart and its latest-block timestamp trailed wall clock by >10
  minutes, so EIP-3009 authorizations signed with the SDK default
  `validAfter = now − 600s` reverted on-chain with `AuthorizationNotYetValid`
  and the facilitator reported `transaction_failed` (with an empty transaction
  hash — it never broadcasts). A payer that backdates `validAfter` further
  (e.g. 2 h) settles fine. This affects any x402 client using SDK defaults,
  including kpass agents, until the chain catches up.
- **pieUSD's `transferWithAuthorization` is non-canonical.** It packs the
  signature as `bytes` instead of `(uint8 v, bytes32 r, bytes32 s)`, so its
  selector is `0xcf092995`, not EIP-3009's `0x927da105`. The facilitator knows
  both; a raw `eth_call` simulation must use the pieUSD form.
- **x402 Go SDK: gin middleware + reverse proxy can mask settlement failure.**
  `httputil.ReverseProxy` streaming triggers gin's `Flush()` →
  `WriteHeaderNow()`, which commits status 200 (and the upstream headers) to the
  client before the middleware settles. If settlement then fails, the intended
  402 + failure `PAYMENT-RESPONSE` header cannot be delivered — the client sees
  `200 {}` while nothing was charged. On settlement success the body is intact
  but the `PAYMENT-RESPONSE` header is lost the same way. Root cause is the
  SDK's `responseCapture` not isolating `Flush` from the wrapped writer; worth
  an upstream issue against `coinbase/x402`. Transaction hashes should be
  verified on-chain rather than trusted from the response header.
