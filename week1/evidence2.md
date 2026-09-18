# Evidence — SPEC02：公网部署 + 真实付费轮（2026-09-18）

> 对应规格 `0917-SPEC02.md`，前置为 SPEC01（本地全链路已验证）。
> 一句话：公网 https 服务上线并通过全部协议断言，真实付费与反向不结算均有链上实证，
> manifest 转正并推送；唯一剩余动作是用户在看板网页提交（材料见 §7）。

## 0. 结论速览

| 项 | 结果 |
|---|---|
| 公网 https 部署 | ✅ Render 免费档 `https://frankfurter-fx.onrender.com`（Docker，Root Dir `services/frankfurter-fx`，`PAY_TO` 为 Secret） |
| 外网复测 | ✅ `/healthz` 200；未付款 402 头解码 4/4 PASS；TLS 有效（raw curl 可达 = Passport 服务端可抓取） |
| 公网真实付费（验收 7） | ✅ 200 + FX body + 结算 tx `0x186f8e20…9336f`（链上确认收款方 == `PAY_TO`） |
| 反向不结算（验收 8） | ✅ 上游 404 → 无结算交易、余额分毫不差 |
| manifest 转正 | ✅ `status: testnet` + `base_url`，`npm run validate` exit 0（commit `98bbff1`） |
| 仓库归属 | ✅ fork `BrucerTao/kite-x402-services`，SSH 推送验证，3 个 commit 已推 |
| 看板提交 | ⏳ 唯一待办：用户网页操作（材料 §7 已备齐） |
| 主网真实结算（Track B） | ⏭️ 用户决定跳过（不花真钱） |

## 1. 部署产物（新增于 `services/frankfurter-fx/`，同步在 `week1/frankfurter-fx/`）

- **Dockerfile**：多阶段交叉编译（`--platform=$BUILDPLATFORM` + `GOARCH=$TARGETARCH` →
  arm64 Mac 也能产出 linux/amd64 镜像）+ `distroless/static`（自带 CA 证书）；
  `CGO_ENABLED=0` 纯静态。本机以等价交叉编译验证：`ELF 64-bit x86-64, statically linked`。
- **render.yaml**：声明式参考（free 档 / docker / `/healthz` 健康检查 / `PAY_TO: sync:false`）。
  ⚠️ Render Blueprint 只读仓库根目录的 render.yaml，实际部署走**面板手动 Web Service**。
- **verify_public.sh**：公网复测脚本——warm-up 重试（免费档冷启动约 50s）→ `/healthz` 200
  断言 → 未付款 402 + base64 解码 4 项 testnet 参数断言 → 输出后续付费命令。
- `.dockerignore`：镜像不含 `.env`/密钥/`.kite-passport`/二进制；配置全部运行时 env 注入。
- 免费档注意：闲置约 15 分钟休眠，每次付费调用前先 `curl <HOST>/healthz` 唤醒。

## 2. 公网复测（verify_public.sh 实测）

```
GET /healthz                      → 200 {"asset":"pieUSD","network":"eip155:2368","ok":true,"price":"$0.001"}
GET /v1/latest?base=USD (未付款)  → 402 + payment-required 头
解码断言 4/4 PASS：network==eip155:2368 / extra.name==pieUSD / extra.version==1
                  / amount==1000000000000000
附加：POST /v1/latest → 402（所有方法均门禁）；/ → 404（无代理泄漏）；TLS verify=0
```

## 3. 公网真实付费（验收 7）——过程与根因

### 3.1 曲折：首次付费「假 200」

首次对公网付费（fxpayer，SDK 默认签名窗口）：`STATUS=200` 但 `BODY={}`、无
`PAYMENT-RESPONSE` 头、**余额纹丝不动**——表面成功、实际未结算。逐层排查：

1. debug 副本加 ErrorHandler → 真因 `settlement failed: transaction_failed`
2. 直连 facilitator（`fxdiag/`）：`/verify` → `isValid:true`；`/settle` →
   `transaction_failed` 且**连交易哈希都没有**（facilitator 未广播，广播前模拟就 revert）
3. `eth_call` 链上模拟 revert，错误数据 `0xdf8e4372` = **`AuthorizationNotYetValid()`**
4. **根因：Kite 测试网链头滞后**——SDK 签名 `validAfter = now−600s`，而实测链最新块
   时间戳落后墙钟约 29 分钟（出块间隔 ~36 分钟）→ `block.timestamp < validAfter` → revert。
   SPEC01 时链头无此滞后故成功，差异在链基础设施，与服务代码无关。

### 3.2 Workaround：fxpayer2

`fxpayer2/`（源码在 `week1/fxpayer2/`）完整复刻 x402 付费流程，仅把 `validAfter` 回拨
600s → **2 小时**。facilitator `/verify` 不限制 validAfter 多旧（只验签名与 validBefore），
链滞后 30 分钟内也能结算。

### 3.3 付费调用与链上实证

```
$ ./fxpayer2 "https://frankfurter-fx.onrender.com/v1/latest?base=USD"
PAYER=0xD2B1…d348   STATUS=200   ELAPSED=3.609s
BODY={"amount":1.0,"base":"USD","date":"2026-09-18","rates":{…31 个币种的 ECB 汇率…}}
```

结算交易（RPC 实证）：

```
tx 0x186f8e2090d80f39b0b83883bf3351e0085368fd6a1ad4c08ca2a5f554b9336f
   block 22041165 @ 2026-09-18 15:03:16 UTC；relayer = facilitator 中继器
   from(payer) 0xD2B1…d348 → to(pay_to) 0x1a83…06d8 (== PAY_TO ✓)
   value 0.001 pieUSD（1000000000000000 / 1e18）
浏览器复核： https://testnet.kitescan.ai/tx/0x186f8e2090d80f39b0b83883bf3351e0085368fd6a1ad4c08ca2a5f554b9336f
```

余额对账：faucet 100.000 − 3×0.001（SPEC01 本地 + 本轮 debug + 本轮公网）= payer **99.997** ✓，
PAY_TO **0.003** ✓（`eth_call`）。

## 4. 反向验证：上游失败不结算（验收 8）

```
$ ./fxpayer2 "https://frankfurter-fx.onrender.com/v1/latest?base=NOTACURRENCY"
STATUS=404  BODY={"message":"not found"}     ← Frankfurter 上游真实 404
```

买方已签名授权、付费流程走到上游，上游返回 404（≥400）→ 中间件**不调用 settle**。
链上复核：PAY_TO 转账仍为 3 笔（无新增），payer 仍 99.997 —— 分毫不差，未扣费 ✓。
（验收标准 2 正反两面都实证。）

## 5. 发现的两个环境级缺陷（对 bounty 有价值，服务代码未改动）

1. **x402 Go SDK（gin middleware × ReverseProxy）**：反向代理流式转发触发 gin `Flush()`
   把 200 提前刷给客户端；此后 settlement 失败时 402 与 `PAYMENT-RESPONSE` 头无法送达
   （客户端看到 200+`{}`），结算成功时该头同样丢失。属 `coinbase/x402` SDK 的
   `responseCapture` 未实现 Flush 隔离，值得提 issue。
2. **Kite 测试网出块停滞/滞后**（~36 min/块，链头落后墙钟 >10 分钟）触发
   `AuthorizationNotYetValid`，SDK 默认 600s 签名窗口的支付全部无法结算；kpass 等标准
   客户端同样会中招。等待链恢复或 Kite 侧修复。

## 6. 安全核对

- 本轮所有上链交易均为 testnet pieUSD（faucet 免费领取），未花任何真实资金
- `git add -n` 干跑 + `git check-ignore` 实证：`.env*`（非 example）/ `*.key` /
  `.kite-passport/` / 二进制均不入库；`render.yaml` 的 `PAY_TO` 为 `sync:false`，值只在面板
- 付款方私钥仅存 `/tmp/fxpayer/payer.key`（0600）；`kite.go`/`main.go` 未改动

## 7. 看板提交材料清单（用户网页操作）

- 方向：`x402-service`；仓库：`https://github.com/BrucerTao/kite-x402-services`（默认分支 main）
- **Commit SHA：`591749e`**（前序：`7a4304b` 服务接入 → `98bbff1` manifest 转正 → `591749e` 收口）
- 公网地址：`https://frankfurter-fx.onrender.com`（`/healthz` 200；未付款 `/v1/latest` 402）
- `service.yaml`：`status: testnet`、`npm run validate` exit 0
- 示例请求/响应：§3.3（200 + ECB 汇率 JSON）与 §4（404 + not found）
- 交易哈希：`0x186f8e2090d80f39b0b83883bf3351e0085368fd6a1ad4c08ca2a5f554b9336f`
  （testnet.kitescan.ai 可查，收款方 == `PAY_TO`）

> 附：kpass 现状 —— `agent:session execute` 为后端服务端抓取，须公网 https；
> 旧会话已 401，复跑需 `kpass login init --email <邮箱>` 重登。过程决策（Render 免费档、
> Track B 跳过、`maintainer.github=BrucerTao`）详见 `0917-SPEC02.md`。
