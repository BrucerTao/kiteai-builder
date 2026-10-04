# Evidence — open-meteo-weather 公网 testnet 全链路验证（2026-10-04）

[1004-SPEC.md](1004-SPEC.md) 的执行结果记录，验证清单已全部勾选。服务摘要见
[services/open-meteo-weather/README.md](services/open-meteo-weather/README.md)；
原始取证文件（未改动的 curl / 脚本输出）复制在本地证据仓
`kiteai-builder/week3/evidence-raw/`，未收录进本仓库。

## 环境与参与者

| 项 | 值 |
|---|---|
| 服务 host | `https://open-meteo-weather-5d7z.onrender.com`（Render 免费档：闲置 ~15min 休眠、冷启动 ~50s，付费调用前先 warm `/healthz`） |
| 部署形态 | Render Web Service，Root Directory `services/open-meteo-weather`，Build `npm ci && npm run build`，Start `node dist/index.js`，Health `/healthz`，`NODE_VERSION=22` |
| network | `eip155:2368`（Kite 测试网；出块头滞后墙钟实测 ~19 分钟） |
| 付费资产 | pieUSD `0x38129cf4CE5E183eFF248F42A7D345Bb1B47621A`（18 位精度；EIP-712 domain `name`/`version` = `pieUSD` / `1`） |
| 价格 | `$0.001`/次 → `maxAmountRequired = 1000000000000000` |
| facilitator | `https://facilitator.pieverse.io/v2` |
| `PAY_TO`（收款方） | `0x1a8374b0F0D1074849e0cA31589532C2ad2806d8` |
| 付款方 | `0xB61D595F91B0303A5804C2c99EF2002E7e0E5e5f`（自建 harness fxpayer3，源码在 `kiteai-builder/week3/fxpayer3/`） |
| 付款资金来源 | 2026-10-04 由 PAY_TO 钱包经 MetaMask 手动转账 0.003 pieUSD（kpass 水龙头与网页水龙头后端全部停摆，见下）。付款私钥只在 `/tmp/fxpayer3/payer.key`（0600），未复制到任何目录、绝不入库 |

## 验证结果

### 1. 本地（`http://localhost:8080`，`node dist/index.js`）

| # | 检查 | 结果 | 原始证据 |
|---|---|---|---|
| 1 | 未付费 `GET /v1/forecast` | 402；`PAYMENT-REQUIRED` 解码 4/4：network `eip155:2368`、`extra.name=pieUSD`、`extra.version=1`、`amount=1000000000000000` | `unpaid_headers.txt`、`unfunded_decoded.json` |
| 2 | 真实付费 `GET /v1/forecast?latitude=52.52&longitude=13.41&current=temperature_2m,wind_speed_10m&hourly=temperature_2m&forecast_days=1` | 200（5.02s）+ Open-Meteo JSON（`temperature_2m=15.5`、`wind_speed_10m=4.0`）；`PAYMENT-RESPONSE` 解码 `success:true`；结算 tx [`0x1fde6d8a…56ab`](https://testnet.kitescan.ai/tx/0x1fde6d8aca3bd9219982e381fb2e53916eb3ab70f454e4d8523035a9ce5056ab)；链上 receipt：status `0x1`、block `22041926`、Transfer `0xB61D…→0x1a83…` 0.001 pieUSD | `local_paid_raw.txt`、`local_paid_receipt.txt` |
| 3 | 反向（上游 400）`latitude=999` | 400 `{"reason":"Latitude must be in range of -90 to 90°. Given: 999.0.","error":true}`；无 `PAYMENT-RESPONSE` 头、无结算交易 | `local_reverse_raw.txt` |

### 2. 公网（`https://open-meteo-weather-5d7z.onrender.com`）

| # | 检查 | 结果 | 原始证据 |
|---|---|---|---|
| 4 | `verify_public.sh <host>` | `/healthz` → 200（`{"ok":true,"network":"eip155:2368","asset":"pieUSD","price":"$0.001"}`）；未付费 402 解码 4/4 → `ALL_PASS` | `verify_public_out.txt` |
| 5 | 真实付费（warm 后同一端点） | 200（3.16s）+ Open-Meteo JSON；`PAYMENT-RESPONSE` 解码 `success:true`；结算 tx [`0xf02b84f5…9798`](https://testnet.kitescan.ai/tx/0xf02b84f59806c886c3d9ede73bbec6caaf08005b802d5e54410dd0eacd539798)；链上 receipt：status `0x1`、block `22041927`、收款方 == `PAY_TO`、0.001 pieUSD | `public_paid_raw.txt`、`public_paid_receipt.txt` |
| 6 | 反向（上游 400）`latitude=999` | 400、无 `PAYMENT-RESPONSE` 头、无新结算交易 | `public_reverse_raw.txt` |

### 3. 余额审计（逐笔核对 pieUSD Transfer 事件）

| 钱包 | 付费前 | 本地付费后 | 公网付费后 | 两次反向请求后 |
|---|---|---|---|---|
| 付款方 `0xB61D…` | 0.003 | 0.002 | 0.001 | 0.001（不变） |
| 收款方 `0x1a83…` | 0.000 | 0.001 | 0.002 | 0.002（不变） |

（付费前付款方的 0.003 由收款方钱包出资转入，出资后收款方余额清零。）
验证窗口内 pieUSD 合约 `0x3812…621A` 上只存在两笔 `payer → PAY_TO` 的
0.001 pieUSD Transfer，即上述两笔结算；两次上游 400 请求前后余额零变动，
未产生任何结算交易。

## 已知环境问题（影响复现方式，不影响上述结论）

- **测试网链头滞后墙钟 ~19 分钟**：EIP-3009 授权若用 SDK 默认「当前时间 − 600s」
  的 `validAfter`，链上会 revert `AuthorizationNotYetValid`；fxpayer3 将
  `validAfter` 回拨 2 小时。facilitator `/verify` 不校验 `validAfter` 新旧，
  只看签名与 `validBefore` 是否在未来。
- **kpass 不能支付自建 host**：`agent:session execute` 只放行 Kite 服务目录内的
  host，自建公网服务被 `host_not_in_executable_catalog` 拒绝；本轮付费走自建
  fxpayer3（week1 的公网付费同样如此）。
- **测试网水龙头全渠道停摆**（2026-10-04 实测）：`kpass faucet`（prod/staging）
  禁用；网页水龙头后端 `/api/sendToken` 对所有 token/chain 组合返回 500。

## 复现

```bash
# 未付款链路（免费，不花资金）
./services/open-meteo-weather/verify_public.sh https://open-meteo-weather-5d7z.onrender.com

# 真实付费链路（需一个已持有 pieUSD 的 EOA；warm 实例后执行）
curl -s "$HOST/healthz"
/tmp/fxpayer3/fxpayer3 "$HOST/v1/forecast?latitude=52.52&longitude=13.41&current=temperature_2m,wind_speed_10m&hourly=temperature_2m&forecast_days=1"
# -> 200 + Open-Meteo body + settle tx；kitescan 收款方应为 PAY_TO
/tmp/fxpayer3/fxpayer3 "$HOST/v1/forecast?latitude=999&longitude=13.41&current=temperature_2m"
# -> 400 且无新结算 tx
```
