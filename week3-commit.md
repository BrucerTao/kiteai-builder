# week3-commit.md — `BrucerTao/kite-x402-services` 提交记录（Week 3）

- **仓库**：<https://github.com/BrucerTao/kite-x402-services>（fork 自 `gokite-ai/kite-x402-services`）
- **分支**：`main`（默认分支，看板只认默认分支的 Commit SHA）
- **基线**：`591749e`（Week 1 收尾状态）
- **本周共 5 个 commit，全部已推送**（`591749e..d2d22bc`）
- **看板提交用最终 HEAD SHA** = **`d2d22bc`**（远端 `main` HEAD，已核验）

| # | SHA | 时间 | 类型 | 一句话 |
|---|---|---|---|---|
| 1 | `81aa544` | 17:24 | fix | 修复 TS 模板 base-path 拼接缺陷（与 Go 模板行为对齐） |
| 2 | `19d327d` | 17:25 | docs | 入库架构文档、参与者指南与 3 份执行规格 |
| 3 | `8e6aa1e` | 17:31 | feat | open-meteo-weather 适配 Render 部署 + 公网免费复测脚本 |
| 4 | `c240080` | 17:54 | feat | 公网部署验证完成，manifest 转 `status: testnet` |
| 5 | `d2d22bc` | 17:55 | docs | evidence.md 全链路证据 + 已知问题同步 |

---

## Commit 1 — `81aa544` fix(template-ts): join upstream base path when proxying /v1 requests

> 2026-10-04 17:24 (+0800) · 3 files changed, **+12/−3**

修复 TS 模板代理拼接：剥掉本服务 `/v1` 前缀后，剩余路径拼接到 `UPSTREAM_URL`
的 base path 上。修复前 `new URL("/forecast", base)` 会整体替换 base 的 path，
`UPSTREAM_URL=.../v1` 命中 `/forecast` → 404，且与 Go 模板（`httputil` 默认
director）行为不一致。

| 文件 | 变更 |
|---|---|
| `templates/typescript-express/src/index.ts` | +4/−1：代理目标改为 `upstream.pathname.replace(/\/+$/, "") + req.originalUrl.replace(/^\/v1/, "")`（`:69` 一带）。`kite.ts` 未动 |
| `templates/typescript-express/.env.example` | ±6：`UPSTREAM_URL` 需含 base path 的说明与示例 |
| `templates/typescript-express/README.md` | +5：写明两个模板共用同一条拼接规则 |

## Commit 2 — `19d327d` docs(repo): add architecture, participant guide and execution specs

> 2026-10-04 17:25 (+0800) · 7 files changed, **+1273/−2**

把此前散在本地工作区的过程/规格文档正式入库。

| 文件 | 行数 | 内容 |
|---|---|---|
| `ARCHITECTURE.md` | +421 | 组件关系、请求生命周期、启动步骤、硬不变量、已知问题（含 kpass 限制与链头滞后） |
| `1004-SPEC.md` | +220 | 本周执行规格（base-path 修复 → 部署 → 付费验证 → 转 testnet） |
| `0917-SPEC01.md` | +192 | Week 1 方法论：本地全链路付费验证 |
| `AGENTS.md` | +185 | 活动提交机制、环境、上游选型约束、执行顺序 |
| `0917-SPEC02.md` | +171 | Week 1 部署规格：Render + 付费验证路径 |
| `PARTICIPANT_GUIDE.md` | +79 | 活动提交机制、审核红线 |
| `README.md` | ±7 | 总览补链接 |

## Commit 3 — `8e6aa1e` feat(services): prepare open-meteo-weather for testnet deploy on Render

> 2026-10-04 17:31 (+0800) · 7 files changed, **+170/−11**

服务同步模板修复（此后与服务 `src` 逐字节一致），manifest 从占位模板转为
testnet 真配置，新增 Render 声明与公网免费复测脚本；状态保持 `draft` 直到付费
验证通过。

| 文件 | 变更 |
|---|---|
| `services/open-meteo-weather/src/index.ts` | +4/−1：同步模板 base-path 修复；此后与模板 diff 逐字节相同 |
| `services/open-meteo-weather/service.yaml` | ±11：`network` eip155:2366→**2368**、`pay_to` 零地址→真实地址、`upstream.url` → `https://api.open-meteo.com/v1`、`maintainer` → BrucerTao |
| `services/open-meteo-weather/.env.example` | +19：`UPSTREAM_URL` 带 `/v1`、testnet 示例 |
| `services/open-meteo-weather/render.yaml` | +27（新增）：native Node 声明（Root Directory 说明、Build/Start、Health `/healthz`、`NODE_VERSION=22`） |
| `services/open-meteo-weather/verify_public.sh` | +108（新增）：公网复测脚本 —— warm 重试 → `/healthz` 200 → 未付费 402 头解码 4/4 断言 |
| `services/open-meteo-weather/README.md` | ±10：部署说明指向 Render |
| `services/README.md` | ±2：目录行更新（draft） |

## Commit 4 — `c240080` feat(services): open-meteo-weather live on public testnet

> 2026-10-04 17:54 (+0800) · 4 files changed, **+31/−25**

Render 部署（`https://open-meteo-weather-5d7z.onrender.com`）后公网验证通过：
未付费 402 解码 4/4、真实付费 200 + 结算 tx、反向 400 不结算；manifest 转正。

| 文件 | 变更 |
|---|---|
| `services/open-meteo-weather/service.yaml` | +3/−1：`status: draft → testnet`；新增 `base_url: https://open-meteo-weather-5d7z.onrender.com` |
| `services/open-meteo-weather/README.md` | +23/−4：付费调用说明（自建 EIP-3009 payer）+ "Verified end-to-end" 表（含公网结算 tx 链接）+ Live 地址 |
| `services/open-meteo-weather/verify_public.sh` | ±28：kpass 示例替换为 fxpayer3 路径 + 环境限制 NOTE（catalog 门禁 / 水龙头停摆） |
| `services/README.md` | ±2：目录行 `draft → testnet` |

## Commit 5 — `d2d22bc` docs(services): record open-meteo-weather testnet evidence and known issues

> 2026-10-04 17:55 (+0800) · 4 files changed, **+96/−15**

| 文件 | 变更 |
|---|---|
| `evidence.md` | +76（新增）：全链路证据 —— 环境/参与者、本地与公网 6 项验证表、逐笔余额审计、已知环境问题、复现命令 |
| `1004-SPEC.md` | ±21：状态头改「已执行完成」；验证清单 9/10 勾选（仅 push 项待勾） |
| `AGENTS.md` | ±6：已知问题改写 —— open-meteo-weather 已上线 testnet |
| `ARCHITECTURE.md` | +8/−2：§9 改写 + 新增「kpass 不能支付自建 host / 水龙头停摆」小节 |

---

## 验收映射（方向 01 `x402-service`）

| 验收标准 | 证据 |
|---|---|
| 1. 未付费 402 + `PAYMENT-REQUIRED`，`accepts[0].network` 正确 | 本地 + 公网解码 4/4 `ALL_PASS`（`eip155:2368` / pieUSD / `1` / `1000000000000000`），见 `evidence.md` §1#1、§2#4 |
| 2. verify → upstream → settle，仅上游 <400 结算 | 反向 `latitude=999` → 上游 400 → 无 `PAYMENT-RESPONSE`、无结算、余额零变动（§1#3、§2#6、§3） |
| 3. `example_request` 实测 2xx | 公网付费调用即 manifest `example_request`（52.52/13.41 + current/hourly + forecast_days=1）→ 200 + Open-Meteo JSON |
| 4. `npm run validate` exit 0 | `✓ 2 service manifest(s) valid`（d2d22bc 工作区实测） |
| 5. 真实付费记录 | 公网结算 tx [`0xf02b84f5…9798`](https://testnet.kitescan.ai/tx/0xf02b84f59806c886c3d9ede73bbec6caaf08005b802d5e54410dd0eacd539798)（block 22041927）；本地 [`0x1fde6d8a…56ab`](https://testnet.kitescan.ai/tx/0x1fde6d8aca3bd9219982e381fb2e53916eb3ab70f454e4d8523035a9ce5056ab)（block 22041926）；收款方均为 `PAY_TO`，0.001 pieUSD/笔 |

交付材料：公网地址 <https://open-meteo-weather-5d7z.onrender.com>、`service.yaml`、
`npm run validate` 结果、示例请求与响应、两笔结算 tx —— 汇总见 `week3/evidence.md`
与 `week3/evidence-raw/`。

---

## 未入库内容（有意排除，非遗漏）

| 内容 | 原因 |
|---|---|
| `services/open-meteo-weather/.env` | 仅含公开值（PAY_TO / KITE_NETWORK / UPSTREAM_URL / PRICE / PORT），纪律要求 `.env` 不入库；Render 面板以环境变量配置。已复制到 `week3/open-meteo-weather/` |
| `/tmp/fxpayer3/payer.key` | 付款方私钥（0600），**永不复制、永不入库** |
| `/tmp/fxpayer3/fxpayer3`、`genkey` 二进制 | 编译产物；源码已同步到 `week3/fxpayer3/`，`go build` 可重建 |
| `services/open-meteo-weather/dist/`、`node_modules/` | 构建产物与依赖 |
| 原始证据 `/tmp/week3-evidence/`（17 个文件） | 未入服务代码仓；已复制到 `week3/evidence-raw/` |

## 中间产物同步说明

| 路径 | 内容 |
|---|---|
| `week3/open-meteo-weather/` | 服务包：11 个入库文件（`.env.example`、`.gitignore`、`README.md`、`package.json`、`package-lock.json`、`render.yaml`、`service.yaml`、`tsconfig.json`、`verify_public.sh`、`src/index.ts`、`src/kite.ts`）+ 本地 `.env`，与 `d2d22bc` 提交状态一致 |
| `week3/fxpayer3/` | 付款 harness 源码：`main.go`、`go.mod`、`go.sum`、`cmd/genkey/main.go`（不含私钥与二进制） |
| `week3/evidence-raw/` | 17 个原始取证文件（curl 输出、402 头、结算 receipts、余额审计与复测脚本输出） |
| `week3/1004-SPEC.md`、`week3/evidence.md` | 与 `d2d22bc` 提交状态一致 |

复测入口（不花资金）：

```bash
./week3/open-meteo-weather/verify_public.sh https://open-meteo-weather-5d7z.onrender.com
```
