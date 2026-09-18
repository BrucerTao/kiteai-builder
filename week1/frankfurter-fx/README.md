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
