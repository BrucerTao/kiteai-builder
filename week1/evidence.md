# Evidence — frankfurter-fx x402 付费服务（Week 1，本地 testnet 全链路）

> 本文件汇总执行 `0917-SPEC01.md` 过程中产生的**证明证据 / 终端输出文本**。
> 所有链上数据可在 Kite testnet 浏览器复核。日期：2026-09-17/18。

## 0. 结论速览

| 判据 | 结果 | 关键证据 |
|---|---|---|
| 未付费 `/v1/latest` → 402 + `PAYMENT-REQUIRED` | ✅ | E2：8/8 断言 PASS |
| verify → upstream → settle，仅上游成功才结算 | ✅ | E8 真实结算 + E9 上游 4xx 不结算 |
| `example_request` 实测 2xx | ✅ | E8：HTTP 200 + 真实汇率 body |
| `npm run validate` exit 0 | ✅ | E6：`✓ 2 service manifest(s) valid` |
| 真实付费调用记录（交易哈希） | ✅ | E3/E8：tx `0xa56ccf24…`，收款方 == `PAY_TO` |
| 主网交易编码逻辑正确（不花真钱） | ✅ | E5：5/5 断言 PASS |
| kpass execute 打 localhost | ❌ 不可行 | E7：后端服务端抓取，回 `invalid request` |

**关键事实**：`kpass agent:session execute` 由 Passport 后端**服务端抓取** merchant URL，且 agent
EVM 私钥**后端托管**，因此**无法**对 `http://localhost` 完成付费调用。本轮真实付费调用改用
**自建本地付款钱包 + x402 Go SDK 客户端**（`fxpayer/`）完成，链上结算与协议流程与 kpass 路径
完全一致。验收第 7 步用 kpass 复跑须等下一轮**公网 https 部署**之后。

## 1. 关键身份与常量

| 项 | 值 |
|---|---|
| 服务 | `frankfurter-fx`（Go 模板 `templates/go-gin`） |
| 上游 | `https://api.frankfurter.dev/v1`（ECB 汇率，免key） |
| 端点 / 价格 | `GET /v1/latest?base=USD`，`PRICE_USD=0.001` |
| 收款方 `PAY_TO` | `0x1a8374b0F0D1074849e0cA31589532C2ad2806d8` |
| 付款方（fxpayer 自建钱包） | `0xD2B1B5D1C45c6313d08e49186E5B4faF4732d348` |
| testnet | chain `eip155:2368`，RPC `https://rpc-testnet.gokite.ai` |
| pieUSD（testnet 稳定币） | `0x38129cf4CE5E183eFF248F42A7D345Bb1B47621A`，18 位精度，EIP-712 name `pieUSD` / version `1` |
| mainnet | chain `eip155:2366`，USDC.e `0x7aB6f3ed87C42eF0aDb67Ed95090f8bF5240149e`，6 位精度，name `Bridged USDC (Kite AI)` / version `2` |
| facilitator | `https://facilitator.pieverse.io/v2` |
| 区块浏览器 | `https://testnet.kitescan.ai` |

## 2. 复现方式

```bash
# 商户（testnet）：在 services/frankfurter-fx 下
set -a && source .env && set +a && go run .          # 监听 :8080

# 付款方（throwaway harness，本目录 fxpayer/）：
cd fxpayer && go build -o /tmp/fxpayer . 
/tmp/fxpayer gen                                      # 生成新钱包，打印地址
kpass sandbox on --base-url https://passport.staging.gokite.ai
kpass faucet drop --recipient <地址> --token pieUSD --base-url https://passport.staging.gokite.ai
/tmp/fxpayer pay "http://localhost:8080/v1/latest?base=USD" eip155:2368
```

> ⚠️ `fxpayer` 的私钥写在 `/tmp/fxpayer/payer.key`（0600），**未复制进本目录**，也永不入库。
> 本目录只有 `fxpayer/main.go`、`go.mod`、`go.sum`（源码，可复现）。

---

## E1. testnet 商户 `/healthz` → 200

```
$ curl -s http://localhost:8080/healthz
{"asset":"pieUSD","network":"eip155:2368","ok":true,"price":"$0.001"}
[http_status=200]
```

## E2. 未付费 `GET /v1/latest?base=USD` → 402 + `PAYMENT-REQUIRED` 头解码

```
$ curl -i "http://localhost:8080/v1/latest?base=USD"   # 无付费头
HTTP/1.1 402 Payment Required
```

`PAYMENT-REQUIRED` 头 base64 解码：

```json
{
  "x402Version": 2,
  "error": "Payment required",
  "resource": {
    "url": "http://localhost:8080/v1/latest?base=USD",
    "description": "ECB foreign-exchange rates wrapped as an x402 paid service on Kite",
    "mimeType": "application/json"
  },
  "accepts": [
    {
      "scheme": "exact",
      "network": "eip155:2368",
      "asset": "0x38129cf4CE5E183eFF248F42A7D345Bb1B47621A",
      "amount": "1000000000000000",
      "payTo": "0x1a8374b0F0D1074849e0cA31589532C2ad2806d8",
      "maxTimeoutSeconds": 60,
      "extra": { "name": "pieUSD", "version": "1" }
    }
  ]
}
```

断言（验收标准 1）：

```
PASS x402Version==2
PASS network==eip155:2368
PASS scheme==exact
PASS extra.name==pieUSD
PASS extra.version==1
PASS amount==1000000000000000        # 0.001 × 10^18
PASS asset==pieUSD addr
PASS payTo==PAY_TO
ALL_PASS= True
```

## E3. 链上复核 Phase 3 结算交易（`eth_getTransactionReceipt`）

```
tx           : 0xa56ccf24bd5880b667214e64b6d80fbaa8c0da2099bc3a69ff0d3b7a8f606a80
status       : 0x1 (0x1 = success)
blockNumber  : 22041128
from(tx)     : 0x12343e649e6b2b2b77649dfab88f103c02f3c78b     # facilitator 提交者（代付 gas）
to(tx)       : 0x38129cf4ce5e183eff248f42a7d345bb1b47621a     # pieUSD 合约（transferWithAuthorization）
--- pieUSD Transfer 事件 ---
from  : 0xd2b1b5d1c45c6313d08e49186e5b4faf4732d348            # 付款方
to    : 0x1a8374b0f0d1074849e0ca31589532c2ad2806d8            # == PAY_TO ✓
units : 1000000000000000 = 0.001 pieUSD                       # == 价格 ✓
```

> 说明：EIP-3009 是 meta-transaction——付款方只签名，**facilitator 提交上链并代付 gas**，所以
> `tx.from` 是 facilitator 的 EOA，而真正的资金流向看 `Transfer` 事件的 `from/to`（付款方 → PAY_TO）。

浏览器：<https://testnet.kitescan.ai/tx/0xa56ccf24bd5880b667214e64b6d80fbaa8c0da2099bc3a69ff0d3b7a8f606a80>

## E4. 付款方余额（精确扣 0.001）

```
$ eth_call pieUSD.balanceOf(0xD2B1…d348)
payer balance = 99.999 pieUSD (units 99999000000000000000)
# 起始 100.0（faucet），仅一次成功付费 0.001 → 99.999
```

## E5. 主网逻辑验证（`KITE_NETWORK=mainnet`，:8090，**未花真钱**）

```
$ curl -s http://localhost:8090/healthz
{"asset":"USDC.e","network":"eip155:2366","ok":true,"price":"$0.001"}
[http_status=200]

$ curl -i "http://localhost:8090/v1/latest?base=USD"   # 未付费
HTTP/1.1 402 Payment Required
```

`accepts[0]` 解码：

```json
{
  "scheme": "exact",
  "network": "eip155:2366",
  "asset": "0x7aB6f3ed87C42eF0aDb67Ed95090f8bF5240149e",
  "amount": "1000",
  "payTo": "0x1a8374b0F0D1074849e0cA31589532C2ad2806d8",
  "maxTimeoutSeconds": 60,
  "extra": { "name": "Bridged USDC (Kite AI)", "version": "2" }
}
```

断言：

```
PASS network==eip155:2366
PASS extra.name==Bridged USDC (Kite AI)
PASS extra.version==2
PASS amount==1000 (0.001 x 10^6)        # USDC.e 6 位精度
PASS asset==USDC.e
ALL_PASS= True
```

> 同一份代码仅切 `KITE_NETWORK`，链 ID / 资产 / EIP-712 domain / 金额精度全部正确——主网交易
> 编码逻辑无误。本轮**不发起主网真实付费**（不消耗真实 USDC.e）。

## E6. `npm run validate`（仓库根，CI 门禁）

```
$ npm run validate
> validate
> node scripts/validate.mjs

✓ 2 service manifest(s) valid
# exit 0
```

## E7. kpass execute 打 localhost → 后端拒绝（服务端抓取证据）

真实 staging 后端响应：

```
$ kpass agent:session execute --url "http://localhost:8080/v1/latest?base=USD" \
      --base-url https://passport.staging.gokite.ai --output json --no-interactive
{
  "_version": "1",
  "error": "invalid request",
  "hint": "The request was invalid. Check the input parameters and try again.",
  "next_command": "",
  "status": "error"
}
```

拦截 CLI 实际发给后端的请求（把 `--base-url` 指向本地抓包服务），证明 **body 不含请求体**，
由后端服务端重建并抓取 merchant URL：

```
METHOD: POST
PATH:   /sandbox/v1/agent/session/execute
HEADERS:
  User-Agent: kpass/1.10.1 darwin-arm64
  Content-Type: application/json
  X-Signature: MEUCIQC-…(请求体完整性签名)
  X-Timestamp: 1789663991
BODY:
{
  "body_hash":    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "headers_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "method":       "POST",
  "nonce":        "9c6060499cd96fd73ba387a0ee404229",
  "session_id":   "agent_session_01a0b044-f8b5-7653-b709-3f07ed70002c",
  "url":          "http://localhost:8080/v1/latest?base=USD"
}
```

且执行后本地 merchant 日志**没有任何来自该 execute 的请求**（连免费的 `/healthz` 用 execute 打
也一样回 `invalid request`）→ 后端根本够不到本机 localhost。又因 `.kite-passport/agent.json`
只有 `agent_token`(JWT)、**无私钥**，本地也无法用 agent 身份自建付款。

**结论**：真实付费调用必须 merchant 公网 https 可达（验收第 7 步留待下一轮部署后用 kpass 复跑）。

## E8. 真实 testnet 付费调用（自建付款方，验收第 7 步本地等价）✅

faucet 领水到付款方：

```
$ kpass faucet drop --recipient 0xD2B1B5D1C45c6313d08e49186E5B4faF4732d348 --token pieUSD \
      --base-url https://passport.staging.gokite.ai --output json
{
  "amount": "100", "asset": "PIEUSD", "chain_id": 2368,
  "recipient": "0xD2B1B5D1C45c6313d08e49186E5B4faF4732d348",
  "status": "success",
  "transaction_hash": "0xba0c94986da253ce674212659dfb1e7e197d0cc51fa7858961e2a448ee0adf32"
}
# eth_call balanceOf → 100.0 pieUSD 已到账
```

付费调用（`fxpayer pay`，内部自动完成 402 → EIP-3009 签名 → 带 `PAYMENT-SIGNATURE` 重试 → 结算）：

```
$ /tmp/fxpayer pay "http://localhost:8080/v1/latest?base=USD" eip155:2368
PAYER_ADDRESS=0xD2B1B5D1C45c6313d08e49186E5B4faF4732d348
TARGET=http://localhost:8080/v1/latest?base=USD
STATUS=200
ELAPSED=3.407s
BODY={"amount":1.0,"base":"USD","date":"2026-09-17","rates":{"AUD":1.4057,"BRL":5.1307,
     "CAD":1.3995,"CHF":0.82449,"CNY":6.7075,"CZK":21.172,"DKK":6.511,"EUR":0.871,
     "GBP":0.74758,"HKD":7.8452,"HUF":316.16,"IDR":17727,"ILS":3.0333,"INR":95.94,
     "ISK":121.77,"JPY":155.69,"KRW":1382.55,"MXN":17.1881,"MYR":4.099,"NOK":9.4286,
     "NZD":1.7414,"PHP":62.718,"PLN":3.7958,"RON":4.5836,"SEK":9.8175,"SGD":1.2753,
     "THB":33.305,"TRY":48.675,"ZAR":16.2525}}
```

商户日志（`402` 挑战 → `200` 含 verify+upstream+settle，耗时 3.38s）：

```
[GIN] 2026-09-18 - 01:02:50 | 402 |     9.034792ms | ::1 | GET "/v1/latest?base=USD"
[GIN] 2026-09-18 - 01:02:54 | 200 |  3.381526958s  | ::1 | GET "/v1/latest?base=USD"
```

链上结算（同 E3）：tx `0xa56ccf24…`，付款方 → `PAY_TO`，0.001 pieUSD，付款方余额 100.0 → 99.999。

## E9. 反向验证：上游 4xx → 不结算（验收第 2 步反证）✅

```
# 上游对非法 base 返回 404
$ curl -s -o /dev/null -w "%{http_code}" "https://api.frankfurter.dev/v1/latest?base=NOTACURRENCY"
404

BLOCK_BEFORE=22041128   BAL_BEFORE(units)=99999000000000000000   # 99.999 pieUSD

# 付费打一个会让上游 4xx 的请求
$ /tmp/fxpayer pay "http://localhost:8080/v1/latest?base=NOTACURRENCY" eip155:2368
STATUS=404
BODY={"message":"not found"}

# 商户日志：402 挑战 → 404（verify+upstream，但不 settle）
[GIN] 2026-09-18 - 01:10:12 | 402 |     299.833µs | ::1 | GET "/v1/latest?base=NOTACURRENCY"
[GIN] 2026-09-18 - 01:10:13 | 404 |  1.864445583s | ::1 | GET "/v1/latest?base=NOTACURRENCY"

# 链上：付款方余额未变，无新增结算
NEW settlement logs after Phase-3 block: 0
BAL_AFTER(units)=99999000000000000000
DEBITED_units = 0   (expect 0)
```

→ 上游返回 4xx 时，买方**未被扣费**，结算被正确跳过。

---

## 3. 本目录（week1/）内容清单

```
week1/
├── 0917-SPEC01.md          # 规格 + 执行结果（含 localhost 修正）
├── evidence.md             # 本文件
├── frankfurter-fx/         # 商户服务源码（Go 模板）
│   ├── main.go             # x402 中间件 + 反向代理 + 仅成功才结算
│   ├── kite.go             # Kite 链常量 / EIP-712 domain（协议根基，未改动）
│   ├── go.mod / go.sum
│   ├── .env.example        # 配置模板（占位）
│   ├── .env                # testnet 运行配置（仅含公开地址，无私钥）
│   ├── .env.mainnet        # mainnet 运行配置（仅切 KITE_NETWORK）
│   ├── .gitignore          # 排除 .env* 与编译产物
│   ├── README.md
│   └── service.yaml        # manifest（status: draft）
└── fxpayer/                # 自建付款方 harness（throwaway，可复现付费调用）
    ├── main.go             # gen / pay 两个模式
    └── go.mod / go.sum
```

**安全说明**：
- `frankfurter-fx` 与 `fxpayer` 的**编译二进制未复制**（体积大、可重建）。
- `fxpayer` 的**私钥 `payer.key` 未复制**（留在 `/tmp/fxpayer/payer.key`，0600）。
- `.env` / `.env.mainnet` 只含 `PAY_TO` 公开地址与配置，**无任何私钥 / 助记词**。
- 源仓库的 `.kite-passport/`（agent JWT）不在本目录内。

## 4. 下一轮待办（需用户决策）

1. `service.yaml` 的 `maintainer.github` 现为占位 `bruce`，提交看板前须换成**与看板绑定的真实 GitHub owner**。
2. 公网 https 部署（VPS / Fly.io / Render / Railway；禁 localhost / http / 自签 / 浏览器验证型隧道）。
3. 部署后外网复测 `/healthz` 200、未付款 `/v1/...` 402；`service.yaml` 改 `status: testnet` + 填 `base_url`。
4. 用 kpass 从**公网 URL** 复跑真实付费调用 + 反向不结算（届时后端可服务端抓取）。
5. 还原源仓库 `templates/typescript-express/package-lock.json` 的无关 npm 改动后再提交。
