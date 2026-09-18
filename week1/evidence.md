# Evidence — SPEC01：本地 testnet 全链路（2026-09-17/18）

> 对应规格 `0917-SPEC01.md`，公网部署轮见 `evidence2.md`。链上数据可在
> [testnet.kitescan.ai](https://testnet.kitescan.ai) 复核。
> 一句话：SPEC01 判据全部达成。唯一例外是 kpass 打不了 localhost（Passport 后端服务端抓取），
> 真实付费改用自建付款方 fxpayer 完成，链上结算与协议流程与 kpass 路径等价——
> 也因此论证了公网 https 部署的必要性（SPEC02 的动机）。

## 0. 结论速览

| 判据 | 结果 | 关键证据 |
|---|---|---|
| 未付费 `/v1/latest` → 402 + `PAYMENT-REQUIRED` | ✅ | §2：8/8 断言 PASS |
| verify → upstream → settle，仅上游成功才结算 | ✅ | §3 真实结算 + §4 上游 4xx 不结算 |
| `example_request` 实测 2xx | ✅ | §3：HTTP 200 + 真实汇率 body |
| `npm run validate` exit 0 | ✅ | `✓ 2 service manifest(s) valid` |
| 真实付费调用（交易哈希） | ✅ | tx `0xa56ccf24…a80`，收款方 == `PAY_TO` |
| 主网交易编码逻辑（不花真钱） | ✅ | 切 `KITE_NETWORK=mainnet` 后 5/5 断言 PASS |
| kpass execute 打 localhost | ❌ 不可行 | 后端服务端抓取，回 `invalid request`（§5） |

## 1. 关键常量

| 项 | 值 |
|---|---|
| 服务 / 上游 | `frankfurter-fx`（Go 模板）；`https://api.frankfurter.dev/v1`（ECB 汇率，免 key） |
| 端点 / 价格 | `GET /v1/latest?base=USD`，`PRICE_USD=0.001` |
| 收款方 `PAY_TO` | `0x1a83…06d8`（完整值见 `service.yaml` / `.env`） |
| 付款方（自建钱包） | `0xD2B1…d348` |
| testnet | chain `eip155:2368`，pieUSD `0x3812…621A`（18 位精度，EIP-712 name `pieUSD`/v`1`） |
| facilitator / 浏览器 | `https://facilitator.pieverse.io/v2`；testnet.kitescan.ai |

## 2. 402 门禁验证（验收标准 1）

```bash
curl -s http://localhost:8080/healthz          # 200 {"asset":"pieUSD","network":"eip155:2368","ok":true,"price":"$0.001"}
curl -i "http://localhost:8080/v1/latest?base=USD"   # 无付费头 → 402 Payment Required
```

`PAYMENT-REQUIRED` 头 base64 解码后断言（8/8 PASS）：

```
x402Version==2   network==eip155:2368   scheme==exact   extra.name==pieUSD
extra.version==1 amount==1000000000000000(0.001×10^18)  asset==pieUSD addr   payTo==PAY_TO
```

## 3. 真实付费调用（验收标准 7，自建付款方路径）

调用链路：`fxpayer pay` 内部自动完成 **GET 402 → EIP-3009 签名 → 带 `PAYMENT-SIGNATURE` 重试 → facilitator verify/settle**。

```bash
# ① faucet 给付款方充值（tx 0xba0c9498…df32，到账 100.0 pieUSD）
kpass faucet drop --recipient <付款方地址> --token pieUSD --base-url https://passport.staging.gokite.ai

# ② 本地启动商户后付费调用
/tmp/fxpayer pay "http://localhost:8080/v1/latest?base=USD" eip155:2368
# → STATUS=200，BODY 为 ECB 真实汇率 JSON（{"amount":1.0,"base":"USD","date":"2026-09-17",...}）
```

商户日志印证两段式：`402 挑战(9ms)` → `200 含 verify+upstream+settle(3.38s)`。

链上结算复核（`eth_getTransactionReceipt`，比截图更强）：

```
tx     0xa56ccf24bd5880b667214e64b6d80fbaa8c0da2099bc3a69ff0d3b7a8f606a80  status=0x1
       block 22041128；tx.from = facilitator 中继器（EIP-3009 meta-tx，代付 gas）
Transfer 事件：from 0xD2B1…d348(付款方) → to 0x1a83…06d8(==PAY_TO ✓)
       units 1000000000000000 = 0.001 pieUSD（== 价格 ✓）
```

余额对账：付款方 100.0 → **99.999** pieUSD（`eth_call balanceOf`），分毫精确。

## 4. 反向验证：上游 4xx → 不结算（验收标准 2 反证）

```bash
/tmp/fxpayer pay "http://localhost:8080/v1/latest?base=NOTACURRENCY" eip155:2368
# → STATUS=404 {"message":"not found"}（Frankfurter 对未知币种的真实 404）
```

商户日志：`402 挑战 → 404`（走了 verify+upstream，**未 settle**）。链上复核：
该请求后**无新增**结算交易，付款方余额仍 99.999 —— 未扣费 ✓。

## 5. kpass 打 localhost 不可行（SPEC01 关键发现）

```
kpass agent:session execute --url "http://localhost:8080/v1/latest?base=USD" …
→ {"status":"error","error":"invalid request"}
```

抓包 CLI 实际请求：body 只含 `url/method/body_hash/headers_hash/session_id/nonce`，
**不含请求体**——由 Passport 后端服务端抓取 merchant URL，够不到本机 localhost；且
`.kite-passport/agent.json` 只有 `agent_token` JWT、无私钥（后端托管）。**结论：真实付费
调用必须公网 https 可达**，kpass 路径留待 SPEC02 部署后复跑。

## 6. 主网编码逻辑验证（不花真钱）

同一份代码切 `.env.mainnet`（仅 `KITE_NETWORK=mainnet`）启动 ：8090，未付款 402 头解码
5/5 断言 PASS：`network==eip155:2366`、`asset==USDC.e`、`extra.name=="Bridged USDC (Kite AI)"`、
`extra.version==2`、`amount==1000`（0.001×10⁶，6 位精度）——链 ID / 资产 / EIP-712 domain /
金额精度全部正确。本轮不发起主网真实付费。

## 7. 目录内容与安全

```
week1/
├── 0917-SPEC01.md / evidence.md     # 本轮规格 + 本证据
├── frankfurter-fx/                  # 商户服务源码（kite.go 未改动，协议根基）
└── fxpayer/                         # 自建付款方 harness（gen / pay 两模式）
```

- 私钥 `payer.key` 只在 `/tmp/fxpayer/`（0600），未复制、未入库
- `.env` / `.env.mainnet` 只含公开地址与配置，无任何私钥；编译二进制不入库
- `npm run validate` exit 0（CI 门禁）

## 8. 遗留待办 → 已全部由 SPEC02 完成

`maintainer.github` 换真实 owner、公网 https 部署、`status: testnet` + `base_url`、
公网真实付费 + 反向不结算、`package-lock.json` 还原——完成情况见 `evidence2.md` §7。
