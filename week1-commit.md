# week1-commit.md — `BrucerTao/kite-x402-services` 提交记录（Week 1）

- **仓库**：<https://github.com/BrucerTao/kite-x402-services>（fork 自 `gokite-ai/kite-x402-services`）
- **分支**：`main`（默认分支，看板只认默认分支的 Commit SHA）
- **基线**：`893a275`（上游初始状态，未改动）
- **共 3 个 commit，全部已推送**；当前 HEAD = **`591749e`**（**看板提交用这个 SHA**）

| # | SHA | 类型 | 一句话 |
|---|---|---|---|
| 1 | `7a4304b` | feat | 新增 frankfurter-fx x402 包装服务（14 文件 +687 行） |
| 2 | `98bbff1` | feat | 公网部署完成，manifest 转 `status: testnet` |
| 3 | `591749e` | docs | 记录公网真实付费验证 + 两个环境已知问题 |

---

## Commit 1 — `7a4304b` feat(services): add frankfurter-fx x402 wrapper

> 2026-09-18 21:58 (+0800) · 14 files changed, **+687 insertions**

Frankfurter（ECB 汇率，免 key）包装为 Kite 测试网上的 x402 付费服务：未付款
`/v1/*` 返回 402 + `PAYMENT-REQUIRED`；仅上游响应 < 400 时按 verify → upstream →
settle 结算。含 Dockerfile、Render 声明文件与公网复测脚本。

| 文件 | 行数 | 内容 |
|---|---|---|
| `services/frankfurter-fx/main.go` | +113 | 服务主体：gin + x402 付费中间件 + `httputil` 反向代理。`/healthz` 免费；`/v1/*` 付费门禁（价格 `$0.001`/次）；转发前剥除 `PAYMENT-SIGNATURE` 头、注入上游凭证；监听 `0.0.0.0:$PORT`（兼容 Render 动态端口） |
| `services/frankfurter-fx/kite.go` | +91 | Kite 链常量与协议根基：testnet `eip155:2368`（pieUSD，18 位精度，EIP-712 name `pieUSD`/v`1`）与 mainnet `eip155:2366`（USDC.e，6 位精度）参数、金额解析器（`$0.001` → 最小单位整数）、facilitator URL。**与模板一字未改** |
| `services/frankfurter-fx/go.mod` / `go.sum` | +212 | 依赖：gin、`coinbase/x402` SDK（v0.0.0-20260316…）、go-ethereum |
| `services/frankfurter-fx/.env.example` | +19 | 配置模板（`PAY_TO` 等占位符，不含真实值） |
| `services/frankfurter-fx/.gitignore` | +4 | 排除 `.env*` 与本地编译产物 |
| `services/frankfurter-fx/Dockerfile` | +16 | 多阶段交叉编译：`--platform=$BUILDPLATFORM golang:1.25` + `GOARCH=$TARGETARCH` → distroless/static 运行时（自带 CA 证书，目标 linux/amd64） |
| `services/frankfurter-fx/.dockerignore` | +12 | 构建上下文排除：`.env*`、`.git`、`*.key`、`.kite-passport/`、二进制等 |
| `services/frankfurter-fx/render.yaml` | +24 | Render Blueprint 声明（free 档、docker runtime、`/healthz` 健康检查、`PAY_TO: sync:false`）。注：Blueprint 只读仓库根目录的 render.yaml，实际部署走面板手动 Web Service，此文件作为声明式参考 |
| `services/frankfurter-fx/verify_public.sh` | +101 | 公网复测脚本：免费档冷启动预热重试 → `/healthz` 200 断言 → 未付款 402 + base64 解码 4 项 testnet 参数断言 → 输出 0x 地址打码 |
| `services/frankfurter-fx/README.md` | +61 | 服务文档（端点/定价/本地运行/kpass 付费示例/部署要求） |
| `services/frankfurter-fx/service.yaml` | +30 | manifest：`schema: 1`、`status: draft`（此 commit 时点）、`network: eip155:2368`、`pay_to` 带引号、上游条款链接、`example_request` |
| `services/README.md` | +1 | 服务目录表加行 `frankfurter-fx | draft | eip155:2368 | GET /v1/latest | @BrucerTao` |
| `.gitignore`（仓库根） | +3 | 排除 `.kite-passport/`（agent JWT）与 `.qoder/`（本地工具目录） |

## Commit 2 — `98bbff1` feat(services): frankfurter-fx live on public testnet

> 2026-09-18 22:37 (+0800) · 2 files changed, +3/−2

Render 免费档部署成功后（`https://frankfurter-fx.onrender.com`），公网复测通过
（`/healthz` 200；未付款 402 头解码 4/4 参数正确），manifest 按状态机转正。

| 文件 | 变更 |
|---|---|
| `services/frankfurter-fx/service.yaml` | `status: draft → testnet`；新增 `base_url: https://frankfurter-fx.onrender.com` |
| `services/README.md` | 目录行 `draft → testnet` |

## Commit 3 — `591749e` docs(services): record testnet paid-call verification and known issues

> 2026-09-18 23:07 (+0800) · 1 file changed, +38

Phase 4/5（公网真实付费 + 反向不结算）完成后的收口文档。

| 文件 | 变更 |
|---|---|
| `services/frankfurter-fx/README.md` | +38 行：① "Verified end-to-end" 表 —— healthz 200 / 未付款 402 参数 / 付费 200 + 结算 tx [`0x186f8e20…9336f`](https://testnet.kitescan.ai/tx/0x186f8e2090d80f39b0b83883bf3351e0085368fd6a1ad4c08ca2a5f554b9336f)（收款方 == `PAY_TO`，0.001 pieUSD）/ `base=NOTACURRENCY` 404 无结算；② "Known environment issues" —— 测试网链头滞后导致 SDK 默认 600s `validAfter` 窗口全部 revert `AuthorizationNotYetValid`；pieUSD 的 `transferWithAuthorization` 为非标准签名布局（selector `0xcf092995`）；x402 Go SDK 的 gin 中间件 × 反向代理 Flush 缺陷会吞掉结算失败的 402 与 `PAYMENT-RESPONSE` 头 |

---

## 未入库内容（有意排除，非遗漏）

| 内容 | 原因 |
|---|---|
| `services/frankfurter-fx/.env` / `.env.mainnet` | 运行配置（仅含公开 `PAY_TO` 地址，无任何私钥），纪律要求 `.env` 不入库；Render 面板以 Secret 形式配置 |
| `/tmp/fxpayer*/payer.key` | 付款方私钥（0600），**永不复制、永不入库** |
| `.kite-passport/`（agent JWT） | 凭证目录，根 `.gitignore` 已排除 |
| `0917-SPEC01.md` / `0917-SPEC02.md` / `AGENTS.md` / `ARCHITECTURE.md` / `PARTICIPANT_GUIDE.md` | 过程/规格文档，属本地工作区与 `kiteai-builder` 证据仓，不属服务代码仓 |
| `templates/typescript-express/package-lock.json` | 会话初期的无关 npm 改动已还原，未提交（当前工作区干净） |

## 中间产物同步说明

上述三个 commit 涉及的**全部服务文件**已同步至
`/Users/huangtao/money/kiteai-builder/week1/frankfurter-fx/`（14 个入库文件 +
本地运行配置 `.env`/`.env.mainnet`，与 `591749e` 提交状态一致）；付费工具源码在
`week1/fxpayer/`（SPEC01 版）与 `week1/fxpayer2/`（宽签名窗口版）。
