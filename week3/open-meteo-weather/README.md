# Open-Meteo Weather

Wraps the free [Open-Meteo](https://open-meteo.com) forecast API as an x402
service on the Kite chain. Built from the
[typescript-express template](../../templates/typescript-express); the only
change is the `.env`.

| | |
|---|---|
| Endpoint | `GET /v1/forecast` → `https://api.open-meteo.com/v1/forecast` |
| Price | $0.001 per call (pieUSD on Kite testnet, the default here) |
| Upstream auth | none |

## Deploy

```bash
npm install
cp .env.example .env     # set PAY_TO to your Kite wallet
npm run build && node dist/index.js
```

Any host that runs Node 22 works (Fly, Render, Cloud Run, a VPS). The service
must be reachable over public https: Kite Passport fetches the URL server-side,
so `localhost` and tunnels that require a browser check will not work.

On Render, create a Web Service with Root Directory `services/open-meteo-weather`;
the bundled [`render.yaml`](render.yaml) documents the native-Node build/start
commands and environment variables. After deploying, run
`./verify_public.sh https://<host>` to re-check `/healthz` and the 402 quote
without spending funds.

## Try it

```bash
curl -i "$BASE_URL/v1/forecast?latitude=52.52&longitude=13.41&current=temperature_2m"
# 402 with a PAYMENT-REQUIRED header until a payment is attached
```

To make a paid call, attach a signed EIP-3009 authorization from an EOA
holding pieUSD on Kite testnet. The harness used for the verification below
backdates `validAfter` because the testnet head lags wall clock (SDK defaults
revert on-chain — see `AGENTS.md`). `kpass agent:session execute` cannot pay
this service: Kite only executes hosts from its own service catalog, and
self-hosted hosts are rejected with `host_not_in_executable_catalog`.

## Verified end-to-end (Kite testnet, 2026-10-04)

| Check | Result |
|---|---|
| `GET /healthz` | 200 |
| Unpaid `GET /v1/forecast` | 402 + `PAYMENT-REQUIRED` (network `eip155:2368`, pieUSD, v1, amount `1000000000000000`) |
| Paid `GET /v1/forecast?latitude=52.52&longitude=13.41&current=temperature_2m,wind_speed_10m&hourly=temperature_2m&forecast_days=1` | 200 + Open-Meteo body (15.5 °C, 4.0 km/h); settle tx [`0xf02b84f5…9798`](https://testnet.kitescan.ai/tx/0xf02b84f59806c886c3d9ede73bbec6caaf08005b802d5e54410dd0eacd539798), payee == `PAY_TO`, 0.001 pieUSD |
| Paid `GET /v1/forecast?latitude=999&longitude=13.41&current=temperature_2m` (upstream 400) | 400 `{"reason":"Latitude must be in range of -90 to 90°. Given: 999.0."}`, **no settlement**, balances unchanged |

Live at `https://open-meteo-weather-5d7z.onrender.com` (Render free tier; sleeps
after ~15 min idle — warm it with `curl <host>/healthz` before a paid call).
