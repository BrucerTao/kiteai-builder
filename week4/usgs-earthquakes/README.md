# USGS Earthquakes

Wraps the [USGS Earthquake Catalog](https://earthquake.usgs.gov/earthquakes/search/)
(FDSN event API) as an x402 service on the Kite chain. Built from the
[typescript-express template](../../templates/typescript-express); `src/` is
byte-identical to the template, so everything service-specific lives in the
config (`.env`, `service.yaml`, `render.yaml`).

Live: <https://usgs-earthquakes.onrender.com>

| | |
|---|---|
| Endpoint | `GET /v1/query` → `https://earthquake.usgs.gov/fdsnws/event/1/query` |
| Price | $0.001 per call (pieUSD on Kite testnet, the default here) |
| Upstream auth | none (no API key, no published rate limit; 20000 events max per query) |

`/v1/count` also works through the same proxy (→ `.../event/1/count`) but is not
declared in [`service.yaml`](service.yaml): every declared endpoint has to have a
paid, tested `example_request`.

## Deploy

```bash
npm install
cp .env.example .env     # set PAY_TO to your Kite wallet
npm run build && node dist/index.js
```

Quote any `.env` value that contains spaces (`SERVICE_DESCRIPTION="..."`). This
file is sourced by the shell (`set -a; . ./.env`), and an unquoted value silently
truncates, so the service falls back to its generic 402 description.

Any host that runs Node 22 works (Fly, Render, Cloud Run, a VPS). The service
must be reachable over public https: Kite Passport fetches the URL server-side,
so `localhost` and tunnels that require a browser check will not work.

On Render, create a Web Service with Root Directory `services/usgs-earthquakes`;
the bundled [`render.yaml`](render.yaml) documents the native-Node build/start
commands and environment variables. After deploying, run
`./verify_public.sh https://<host>` to re-check `/healthz` and the 402 quote
without spending funds.

## Try it

```bash
curl -i "$BASE_URL/v1/query?format=geojson&starttime=2026-10-01&endtime=2026-10-07&minmagnitude=5.5&limit=5&orderby=magnitude"
# 402 with a PAYMENT-REQUIRED header until a payment is attached
```

To make a paid call, attach a signed EIP-3009 authorization from an EOA holding
pieUSD on Kite testnet. The harness used for the verification below backdates
`validAfter` because the testnet head lags wall clock (SDK defaults revert
on-chain — see `AGENTS.md`). `kpass agent:session execute` cannot pay this
service: Kite only executes hosts from its own service catalog, and self-hosted
hosts are rejected with `host_not_in_executable_catalog`.

### Do not pass `jsonerror=true`

The settlement rule is "charge only if the upstream status is < 400". With
`jsonerror=true` the USGS API reports parameter errors as **HTTP 200** with
`metadata.status: 400` and `title: "Search Error"`, so a failed lookup would be
charged. Bad parameters without that flag return a real 400 and cost nothing.

## Verified end-to-end (Kite testnet, 2026-10-11)

| Check | Result |
|---|---|
| `src/` vs template `src/` | `diff -r` empty (byte-identical) |
| Base-path join (`index.ts:69`) | `/v1/query?…` → `https://earthquake.usgs.gov/fdsnws/event/1/query?…`, query preserved |
| `GET /healthz` (local + public) | 200 `{ok:true, network:"eip155:2368", asset:"pieUSD", price:"$0.001"}` |
| Unpaid `GET /v1/query` (local + public) | 402 + `PAYMENT-REQUIRED`; decoded 5/5 — `eip155:2368`, pieUSD, v1, `1000000000000000`, and `resource.description` is the USGS one |
| Paid `GET /v1/query` (public, the manifest `example_request`) | 200 in 4.4 s + GeoJSON `FeatureCollection` (`metadata.status:200`, `api:2.8.1`, 4 features M5.5–M5.9); settle tx [`0x9c66a3bd…b5ea`](https://testnet.kitescan.ai/tx/0x9c66a3bddafd5c0160cdd91e56f80603f8ff8360450b7222c2eb9f4f3145b5ea), block 22042202, status `0x1`, Transfer payer → `PAY_TO` 0.001 pieUSD |
| Paid `GET /v1/query` (local, same request) | 200 in 7.8 s; settle tx [`0xfbc83dfd…84a7`](https://testnet.kitescan.ai/tx/0xfbc83dfd2ce2c4999ddbf204ad49a857d62d216e98c66fec9963d1ad76cf84a7), block 22042203, status `0x1`, same 0.001 pieUSD Transfer |
| `starttime=not-a-date` (upstream 400) | 400 `Bad starttime value "not-a-date"`, **no `Payment-Response`, no settlement**, balances unchanged |
| Final balances | payer 0.000 / `PAY_TO` 0.002 — both payments returned to the payee, net zero, nothing stranded |
| `npm run validate` (repo root) | exit 0, `✓ 3 service manifest(s) valid` |

Live at `https://usgs-earthquakes.onrender.com` (Render free tier; sleeps after
~15 min idle — warm it with `curl <host>/healthz` before a paid call). Full run
logs: [`evidence-usgs.md`](../../evidence-usgs.md).

## Data credit

Earthquake data is courtesy of the
[U.S. Geological Survey](https://www.usgs.gov/information-policies-and-instructions/acknowledging-or-crediting-usgs).
Most USGS information resides in the public domain and may be used without
restriction; credit is requested, which is what this section and the upstream
`terms_url` in the manifest are for.
