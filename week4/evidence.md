# Evidence — usgs-earthquakes 公网 testnet 全链路验证（2026-10-11）

[1011-SPEC.md](1011-SPEC.md) 的执行结果记录，验证清单已全部勾选（Phase 0–6）。服务摘要见
[services/usgs-earthquakes/README.md](services/usgs-earthquakes/README.md)；原始取证文件
（未改动的 curl / 脚本输出，付费签名已抹除）复制在本地证据仓
`kiteai-builder/week4/evidence-raw/`，未收录进本仓库。

## 环境与参与者

| 项 | 值 |
|---|---|
| 服务 host | `https://usgs-earthquakes.onrender.com`（Render 免费档：闲置 ~15min 休眠、冷启动 ~50s，付费调用前先 warm `/healthz`） |
| 部署形态 | Render Web Service，Root Directory `services/usgs-earthquakes`，Build `npm ci && npm run build`，Start `node dist/index.js`，Health `/healthz`，`NODE_VERSION=22`，`PAY_TO` 设为 Secret，**未设 `PORT`** |
| network | `eip155:2368`（Kite 测试网） |
| 上游 | USGS FDSN event API `https://earthquake.usgs.gov/fdsnws/event/1`（`api: 2.8.1`，无 key、无公开速率限制、公共领域条款） |
| 付费资产 | pieUSD `0x38129cf4CE5E183eFF248F42A7D345Bb1B47621A`（18 位精度；EIP-712 domain `name`/`version` = `pieUSD` / `1`） |
| 价格 | `$0.001`/次 → `amount = 1000000000000000` |
| facilitator | `https://facilitator.pieverse.io/v2` |
| `PAY_TO`（收款方） | `0x1a8374b0F0D1074849e0cA31589532C2ad2806d8`（与另两个服务同一钱包） |
| 付款方 | `0x010F7992CE7806c291d1F494e44EF1840B7e4584`（自建 harness **fxpayer4**；源码在 `kiteai-builder/week4/fxpayer4/`） |
| 付款资金来源 | 2026-10-11 由 `PAY_TO` 钱包经 MetaMask 手动转入 **0.002 pieUSD**（kpass 水龙头与网页水龙头后端全部停摆，见「已知环境问题」）。付款私钥只在 `~/.kite-x402/fxpayer4.key`（0600，仓库外 + `/tmp` 外），未复制到任何目录、绝不入库 |

## 验证结果

### 1. 上游预检（直连 USGS，不花钱）— 全部通过

| # | 检查 | 结果 | 原始证据 |
|---|---|---|---|
| 1 | `example_request` 实测 | `GET /query?format=geojson&starttime=2026-10-01&endtime=2026-10-07&minmagnitude=5.5&limit=5&orderby=magnitude` → **200**，3565 B，0.93 s，`metadata.status=200 api=2.8.1`，4 个 feature（M5.9 印尼 Tambolaka / M5.8 俄 Vilyuchinsk / M5.8 Severo-Kuril'sk / M5.5 阿拉斯加 Aleneva） | `01-upstream-preflight.txt` |
| 2 | 反向触发器（4 个） | `starttime=not-a-date`、`format=banana`、`limit=99999`、未知参数 `zzz=1` → 全部 **400** `text/plain`（`Bad starttime value` / `Bad format value` / `0 <= limit <= 20000` / `Unknown parameter "zzz"`） | 同上 |
| 3 | **jsonerror 陷阱** | `starttime=not-a-date&jsonerror=true` 与 `limit=99999&jsonerror=true` → **HTTP 200** + `metadata.status:400`、`title:"Search Error"`。本 wrapper「上游 < 400 才结算」，所以这种失败查询**会被收费** → 已写进 README 警告与 manifest `pitfalls` 第一条；不改代码（改则破坏「src 与模板逐字节相同」或触发两模板同步义务） | 同上 |
| 4 | `/count` 端点 | **200** `{"count":4,"maxAllowed":20000}` —— 经同一代理可用，但**不写进 manifest**（声明的端点必须有付费实测的 `example_request`） | 同上 |

### 2. 本地（`http://localhost:8091` / `:8092`）

| # | 检查 | 结果 | 原始证据 |
|---|---|---|---|
| 5 | 「配置而非分叉」 | `diff -r templates/typescript-express/src services/usgs-earthquakes/src` → **无输出**（逐字节相同，含 `kite.ts`）；`npm ci` + `npm run typecheck` 通过 | `02-local-healthz-and-402.txt` |
| 6 | base-path 拼接（`index.ts:69`） | `/v1/query?…` → `https://earthquake.usgs.gov/fdsnws/event/1/query?…`，query 完整保留；`/v1/count` 同理 → **PASS** | 同上 |
| 7 | `GET /healthz` | **200** `{"ok":true,"network":"eip155:2368","asset":"pieUSD","price":"$0.001"}`；启动行 `kite x402 service on :8091 -> https://earthquake.usgs.gov/fdsnws/event/1 (network eip155:2368, $0.001 per call to 0x1a83…06d8)` | `02-local-healthz-and-402.txt` |
| 8 | 未付费 `GET /v1/query` | **402** + `PAYMENT-REQUIRED`；base64 解码断言 **4/4 PASS**（`network=eip155:2368`、`extra.name=pieUSD`、`extra.version=1`、`amount=1000000000000000`），`resource.description` = `USGS earthquake catalog queries behind x402 on Kite` | `03`–`06` |
| 9 | 构建产物冒烟（= Render 上的 Build/Start 命令） | `npm run build` → `dist/index.js` + `dist/kite.js`；`node dist/index.js` 在 `:8092` → healthz 200 + 未付费 402 解码 **4/4 PASS** | `09-build-smoke.txt` |
| 10 | 缺钱时的行为（付费前实测） | fxpayer4 签出合法 EIP-3009 授权后重试 → 服务返回 **402**、`PAYMENT-REQUIRED` 的 `error` = **`insufficient_funds`**，1.354 s，**未触达上游、未扣费** | `07-local-harness-unfunded-smoke.txt` |
| 11 | **真实付费** `GET /v1/query`（manifest `example_request`） | **200**（7.813 s）+ GeoJSON `FeatureCollection`；`PAYMENT-RESPONSE` 解码 `success:true`；结算 tx [`0xfbc83dfd…84a7`](https://testnet.kitescan.ai/tx/0xfbc83dfd2ce2c4999ddbf204ad49a857d62d216e98c66fec9963d1ad76cf84a7)；链上 receipt：status **`0x1`**、block **22042203**、gasUsed 66322、Transfer `0x010F…4584 → 0x1a83…06d8` **0.001 pieUSD** | `17-local-paid.txt`、`18-local-settle-and-final-balances.txt` |

### 3. 公网（`https://usgs-earthquakes.onrender.com`）

| # | 检查 | 结果 | 原始证据 |
|---|---|---|---|
| 12 | `GET /healthz` | **200**（首次尝试即通，实例已 warm）`{"ok":true,"network":"eip155:2368","asset":"pieUSD","price":"$0.001"}` → 确认 `KITE_NETWORK` 生效（该变量默认值是 `mainnet`，漏填会静默报主网价） | `11-public-healthz.txt` |
| 13 | `verify_public.sh <host>` | 未付费 `/v1/query` → **402**，解码断言 **5/5 PASS** → `ALL_PASS`。第 5 条 `resource.description is the USGS one` 是本轮新增，用于识破「端口被别的服务占用」 | `12-verify-public.txt` |
| 14 | **真实付费** `GET /v1/query`（manifest `example_request`，**验收标准 5**） | **200**（4.4 s）+ GeoJSON `FeatureCollection`（`metadata.status:200`、`api:2.8.1`、4 个 feature M5.5–M5.9）；`PAYMENT-RESPONSE` 解码 `success:true`、`network:eip155:2368`；结算 tx [`0x9c66a3bd…b5ea`](https://testnet.kitescan.ai/tx/0x9c66a3bddafd5c0160cdd91e56f80603f8ff8360450b7222c2eb9f4f3145b5ea)；链上 receipt：status **`0x1`**、block **22042202**、gasUsed 83398、`to` = pieUSD 合约、标准 ERC-20 Transfer `0x010F…4584 → 0x1a83…06d8`（**== `PAY_TO`**）`0x38d7ea4c68000` = **0.001 pieUSD** | `14-public-paid.txt`、`16-settle-receipt.txt`、`16b-settle-logs.txt` |
| 15 | **反向**（上游 400）`starttime=not-a-date&limit=5`（**验收标准 2 反证**） | **400**（2.067 s）`text/plain` `Bad starttime value "not-a-date". Valid values are ISO-8601 timestamps.`（`Service version: 2.8.1`）；**无 `Payment-Response` 头、无 `SETTLE_RESPONSE`、无结算交易** | `15-public-reverse.txt` |
| 16 | manifest 校验 | 仓库根 `npm run validate` → exit 0，`✓ 3 service manifest(s) valid` | `10-validate.txt` |

### 4. 余额审计（pieUSD，18 位精度，RPC `eth_call balanceOf`）

| 钱包 | 轮初 | 用户转入 0.002 后（13:41:27Z 实测） | 两笔付费后（13:49:07Z 实测） |
|---|---|---|---|
| 付款方 `0x010F…4584` | 0.000000 | **0.002000** | **0.000000** |
| 收款方 `0x1a83…06d8` | 0.002000 | **0.000000** | **0.002000** |

两次采样之间的变化由两笔结算交易的 Transfer 日志逐笔证明（各 0.001，见 #11、#14），
**净变化为零**，付款方余额归零、未留沉没资金（上周 0.001 因私钥放 `/tmp` 被清理而沉没，
本轮刻意把余额花干净）。反向请求（#15）前后余额零变动，未产生任何结算交易。

| 钱包 | 状态 |
|---|---|
| 上周付款方 `0xB61D…5e5f`（fxpayer3） | 0.001000，**永久沉没** —— 私钥只在 `/tmp/fxpayer3/payer.key`，已随 `/tmp` 清理丢失 |

## 本轮新发现

### 链的出块节奏：「滞后」这个数字会误导人

付费前采样发现出块极慢且不均匀，一度以为链停了：

| 区间 | 间隔 |
|---|---|
| block 22042196 → 22042197 | 7.4 min |
| 22042197 → 22042198 | **63 min** |
| 22042198 → 22042199 | 16.5 min |
| 22042199 → 22042200 | 12.9 min |

且**每个块都只含 1 笔交易**。但两笔结算实际都在提交后约 1 分钟内就被打包：

| 结算 | 提交时刻 |  receipt 可查 | 所在块 |
|---|---|---|---|
| 公网 `0x9c66a3bd…b5ea` | 13:46:10Z | 13:47:04Z（54 s） | 22042202 |
| 本地 `0xfbc83dfd…84a7` | ~13:48Z | 13:49:07Z | 22042203 |

**结论**：这条测试网是**有交易才出块**，空闲时的 `head_ts` 落后墙钟几十分钟属正常，
**不代表链停摆，也不代表结算会卡住**。因此：① 不要因为 lag 数字大而推迟付费调用；
② 但 `validAfter` 仍必须回拨（本轮 `BACKDATE_SECONDS=7200`，lag 32 min 时足够），
否则链上按块时间校验会 revert `AuthorizationNotYetValid`；③ facilitator **提交即返回**、
不等确认（HTTP 往返 4.4 s / 7.8 s），所以 receipt 要另行轮询。

### EIP-3009 授权的可重放性只取决于「窗口 + 余额」

取证 #10（付款方缺钱）时抓下了一笔完整的授权 + 签名，当时被 facilitator 拒
（`insufficient_funds`），看起来是废签名。但它的 `validBefore` = `1791725095`
（13:24:55Z）**在用户于 ~13:35Z 给该钱包充值后仍然开着** —— 一笔「因缺钱而失败」的授权，
在钱包到账后就变成了**可重放的活凭证**。已在归档前把 `signature` 字段抹成
`0x<REDACTED>`（`07-local-harness-unfunded-smoke.txt` 文件头写明原因）。

**纪律**：把 harness 的 `PAYLOAD` / `PAYMENT-SIGNATURE` 存进证据文件时，要么当场抹掉签名，
要么确认 `validBefore` 已过 —— 与「上一次是否失败」无关。（本轮即使被重放，资金也只是从
自己的付款方转到自己的 `PAY_TO`，损失有限，但没必要留。）

## 已知环境问题（影响复现方式，不影响上述结论）

- **测试网水龙头全渠道停摆**（2026-10-04 起，本轮复核仍如此）：`kpass faucet`（prod/staging）
  报 "faucet is disabled"；网页水龙头后端 `/api/sendToken` 对所有 token/chain 组合返回 500。
  **新钱包领不到 pieUSD**，资金只能从已有余额的钱包转。
- **kpass 不能支付自建 host**：`agent:session execute` 只放行 Kite 服务目录内的 host，
  自建公网服务报 `host_not_in_executable_catalog`。因此端到端付费验证只能走自建 payer harness。
- **付款方私钥放 `/tmp` 会丢**：上周 0.001 pieUSD 因此沉没 → 本轮改放 `~/.kite-x402/fxpayer4.key`
  （0600，仓库外 + `/tmp` 外），并在轮末把余额花到 0。
- **`.env` 里带空格的值必须加引号**：`.env` 由 shell 读取（`set -a; . ./.env`，应用本身不加载
  dotenv），未加引号的 `SERVICE_DESCRIPTION=USGS earthquake …` 会被截断、变量丢失，402 里只剩
  兜底文案 `Paid API wrapped for the Kite network`。同样的 bug 存在于
  `services/open-meteo-weather/.env:5`；`frankfurter-fx` 的写法（带引号）才是对的。
  **只影响本地开发** —— Render 上的 env 由面板注入（键值对，不经 shell，**不要自己加引号**）。
- **本地验证前必须清场端口**：本轮第一次「本地验证」的请求全被一个 2026-10-04 启动、
  连续占用 `:8080` 七天的 `node dist/index.js`（cwd 是上周的 `open-meteo-weather`）接走。
  `/healthz` 与 402 的 network/asset/amount 两个服务完全一样，看不出异常，只有
  `resource.description` 是上周服务的兜底文案才露出马脚。对策：验证前
  `lsof -nP -iTCP:<port> -sTCP:LISTEN`，并**断言启动日志里的上游地址**；
  `verify_public.sh` 因此新增第 5 条断言。验证完**立刻 kill**（本轮已执行，`:8091` 已释放）。

## 复现

```bash
# 免费部分（不花资金）
./services/usgs-earthquakes/verify_public.sh https://usgs-earthquakes.onrender.com

# 真实付费（需一个已持有 pieUSD 的 EOA；免费档先 warm）
curl -s https://usgs-earthquakes.onrender.com/healthz          # 等 200
BACKDATE_SECONDS=7200 /tmp/fxpayer4/fxpayer4 \
  "https://usgs-earthquakes.onrender.com/v1/query?format=geojson&starttime=2026-10-01&endtime=2026-10-07&minmagnitude=5.5&limit=5&orderby=magnitude"
# -> 200 + GeoJSON + settle tx；kitescan 上收款方应为 PAY_TO、0.001 pieUSD
BACKDATE_SECONDS=7200 /tmp/fxpayer4/fxpayer4 \
  "https://usgs-earthquakes.onrender.com/v1/query?format=geojson&starttime=not-a-date&limit=5"
# -> 400 且无新结算 tx、无 PAYMENT-RESPONSE 头
```

harness 从 `kiteai-builder/week4/fxpayer4/` 重建：`go build -o /tmp/fxpayer4/fxpayer4 .`；
新付款方密钥用 `cmd/genkey`（默认写 `~/.kite-x402/fxpayer4.key`，0600）。
