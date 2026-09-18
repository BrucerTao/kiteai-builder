# Evidence 2 — frankfurter-fx 公网部署轮（`0917-SPEC02.md` 执行情况）

> 对应 `0917-SPEC02.md`。日期：2026-09-18。
> 本文所有链上地址均已脱敏（如 `0x1a83...06d8`）；真实值只在 `service.yaml` / `.env` / 部署平台 secrets，不写进本文件。
> 本轮**未**提交 git、**未**部署公网、**未**花费任何资金、**未**发起任何上链交易。

## 0. 结论速览

| 项 | 状态 | 证据 |
|---|---|---|
| 部署产物 `Dockerfile` / `.dockerignore` / `render.yaml` / `verify_public.sh` | ✅ 已写 | §2 |
| `go build ./...` + `go vet ./...` | ✅ PASS | §3 |
| Dockerfile 构建阶段（linux/amd64 交叉编译） | ✅ 产出 ELF x86-64 静态 | §3 |
| 本地产物运行：`/healthz` 200 + 未付款 `/v1/latest` 402 解码 | ✅ 4/4 ALL_PASS | §4 |
| 公网 https 部署（Render 免费版） | ✅ **完成**：`https://frankfurter-fx.onrender.com` 部署 success，Phase 3 复测 4/4 ALL_PASS | §5、§11 |
| 公网真实付费调用（验收 7） | ✅ **完成**：200 + FX body + settle tx `0x186f8e20…9336f`（链上确认收款方==PAY_TO） | §12 |
| 反向验证：上游失败不结算（验收 8） | ✅ **完成**：404 + 无结算交易 + 余额分毫不差 | §12 |
| `service.yaml` → `status: testnet` + `base_url` | ✅ 完成（commit `98bbff1` 已推送） | §11 |
| 仓库归属（`origin` / `maintainer.github` / git email） | ✅ 完成：fork `BrucerTao/kite-x402-services`（SSH 验证） | §10 |
| 看板提交 | ⏳ 唯一待办：用户网页操作（材料已备齐，见 §12 提交清单） | §12 |
| Track B 主网真实结算（`status: live`） | ⛔ 用户已决定跳过（不花真钱） | §5 |

**一句话**：本轮能在本地、无需授权完成的都已完成并验证（部署产物 + 构建 + 运行期 402 行为）；
公网部署及其后的验收/看板步骤**本质上需要用户本人的平台账号、交互式登录、本人 GitHub 仓库、以及真实
付费/上链授权**，agent 无法自主完成，也**不应**擅自执行（SPEC02 纪律：未经确认不 push / 不部署付费
平台 / 不发起主网结算）。本机实测：`docker` 未安装、`fly`/`flyctl` 未安装、`kpass 1.10.1` 已装但无公网
URL 可打、`go 1.27.1` 可用。

## 1. 目标与本轮边界

SPEC02 目标：把 SPEC01 已本地验证的 `frankfurter-fx` 部署到公网 https → 外网复测 → 用 kpass 从公网 URL
跑真实 testnet 付费 + 反向不结算 → 更新 manifest（`testnet` + `base_url`）→ 修仓库归属 → 提看板；主网真实
结算为可选 Track B。

- **可自主完成（本轮已做）**：写部署产物、验证构建、本地运行期验证 402 行为、产出证据、拷贝到 week1。
- **必须用户参与（本轮阻塞）**：第三方平台账号/登录、本人仓库与身份、真实付费/上链、看板网页提交、主网出资。

## 2. 已实现：部署产物（新增于 `services/frankfurter-fx/`）

### `Dockerfile`
```dockerfile
# ---- build stage ----
# Build on the native platform, cross-compile to the deploy target (Render/Fly shared CPU = linux/amd64).
FROM --platform=$BUILDPLATFORM golang:1.25 AS build
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o /out/frankfurter-fx .

# ---- run stage ----
# distroless/static bundles CA certs, required for TLS to upstream / facilitator / RPC.
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/frankfurter-fx /frankfurter-fx
EXPOSE 8080
ENTRYPOINT ["/frankfurter-fx"]
```
要点：`--platform=$BUILDPLATFORM` + `ARG TARGETARCH` → 在 arm64 Mac 上也能交叉编译出 **linux/amd64**（Render
free / Fly shared CPU 的目标架构）可运行的镜像（规避「本机构建出 arm64 镜像、amd64 平台跑不起来」的坑）；
`distroless/static` 自带 CA 证书（对 upstream / facilitator / RPC 发 TLS 必需）；`CGO_ENABLED=0` 纯静态，契合
static 基础镜像。应用绑 `0.0.0.0:$PORT`（`main.go` 实测 `addr := ":" + env("PORT","8080")`），契合 Render 自动注入 `PORT`。

### `.dockerignore`
```
.git
.env
.env.*
.kite-passport/
*.key
/frankfurter-fx
Dockerfile
.dockerignore
render.yaml
README.md
service.yaml
verify_public.sh
```
要点：镜像**不含**任何 `.env` / 密钥 / `.kite-passport` / manifest / 部署配置 / 本机构建的二进制；配置全部
运行时经 env 注入。

### `render.yaml`（Render 声明式参考配置，免费档；实际部署走手动 Web Service）
> ⚠️ **注意**：Render Blueprint 只读**仓库根目录**的 `render.yaml`，而本文件随服务放在
> `services/frankfurter-fx/` → Blueprint 用不了。实际部署走**手动 Web Service**（SPEC02 Phase 2 步骤 7-8）：
> Root Directory=`services/frankfurter-fx` + Runtime Docker + Free + Health Check `/healthz` + 5 个 env
> 手动录入（`PAY_TO` 设 Secret）。本文件作为参数的声明式参考（一一对应），保持入库。
```yaml
services:
  - type: web
    name: frankfurter-fx
    runtime: docker
    plan: free
    region: oregon
    dockerfilePath: ./Dockerfile
    healthCheckPath: /healthz
    envVars:
      - key: KITE_NETWORK
        value: testnet
      - key: UPSTREAM_URL
        value: https://api.frankfurter.dev/v1
      - key: PRICE_USD
        value: '0.001'
      - key: SERVICE_DESCRIPTION
        value: ECB foreign-exchange rates wrapped as an x402 paid service on Kite
      - key: PAY_TO
        sync: false
```
要点：`plan: free`（免费档）；`runtime: docker` 复用现有 `Dockerfile`；`healthCheckPath: /healthz`（避开探 `/`
非 200 的坑）；**不设 `PORT`**（Render 自动注入，`main.go` 绑 `0.0.0.0:$PORT`）；**`PAY_TO` 用 `sync:false`** →
值在面板填、**不入库**。部署时在面板把 **Root Directory 设 `services/frankfurter-fx`**。
> ⚠️ 免费档闲置约 15 分钟休眠、冷启动约 50s → 每次 kpass 付费调用前先 `curl <HOST>/healthz` 唤醒。

### `verify_public.sh`（公网复测脚本，平台无关；取代原 Fly 专用 `deploy.sh`）
`./verify_public.sh <HOST>` 一条命令完成 Phase 3 复测并打印 Phase 4/5 命令：
1. **warm-up 重试**（免费档冷启动）：循环 `curl /healthz` 至 200（最多 ~90s）；
2. 断言 `GET /healthz` → 200；
3. 断言未付款 `GET /v1/latest?base=USD` → 402，内联 python 解码 `Payment-Required` 并断言 4 项 testnet
   参数（network/pieUSD/version/amount），与本地 dry-run 同判据；
4. 打印**已核实**的 kpass 1.10.1 命令（Phase 4 真实付费 / Phase 5 反向不结算），`<HOST>` 已填好，并提示
   每次 execute 前先 warm。

实测：`bash -n verify_public.sh` 语法通过；PII 扫描无 email/username/完整地址；脚本**不花钱、不发起付费
调用、不触碰任何平台凭据**（只读 `<HOST>` 参数做公网 GET）。
> 局限：尚无公网 `<HOST>`（Render 未部署），未做端到端真实复测；该脚本在有 `<HOST>` 后按上述顺序执行。
> 本轮**移除**了 Fly 专用的 `fly.toml` / `deploy.sh`（Fly 已改按量付费、需绑卡，不符合「免费」要求）。

## 3. 构建验证（本机 `docker` 缺失，用等价交叉编译证明构建阶段）

```
go version go1.27.1 darwin/arm64
go build ./...                                   -> BUILD OK
go vet ./...                                     -> VET OK
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/fx-linux .   -> CROSS-BUILD OK
file /tmp/fx-linux -> ELF 64-bit LSB executable, x86-64, version 1 (SYSV),
                      statically linked, Go BuildID=..., with debug_info, not stripped
```
- 交叉编译产物是 **x86-64 静态 ELF** == Fly `shared`/amd64 目标架构；与 Dockerfile 的
  `RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build` 完全等价（该命令是镜像构建中唯一可能失败的
  编译步骤，已证明通过）。
- 运行阶段只是 `COPY` 二进制 + `ENTRYPOINT`，无编译风险。
- 局限：本机无 `docker`，未跑完整 `docker build`（拉基础镜像 + 组装）；该步需在有 docker/平台的机器上执行，
  预期无阻塞（纯标准多阶段构建）。

## 4. 运行期验证（本地 dry-run，testnet 配置）

启动本地二进制（testnet `.env`：`KITE_NETWORK=testnet`、`UPSTREAM_URL=https://api.frankfurter.dev/v1`、
`PRICE_USD=0.001`、`PORT=8080`）：

```
GET /healthz                         -> HTTP 200
  body: {"asset":"pieUSD","network":"eip155:2368","ok":true,"price":"$0.001"}

GET /v1/latest?base=USD (未付款)     -> HTTP 402 Payment Required
  header: Payment-Required: <base64>
```

`Payment-Required` 头 base64 解码（**地址已脱敏**）：
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
      "asset": "0x3812...621A",
      "amount": "1000000000000000",
      "payTo": "0x1a83...06d8",
      "maxTimeoutSeconds": 60,
      "extra": { "name": "pieUSD", "version": "1" }
    }
  ]
}
```
断言：
```
PASS - accepts[0].network == eip155:2368
PASS - extra.name == pieUSD
PASS - extra.version == 1
PASS - amount == 1000000000000000      # 0.001 x 10^18
RESULT: ALL_PASS
```
- 服务验证完毕后已停止，`port 8080 free`（无遗留监听进程）。
- 注：`resource.url` 显示 `http://localhost:8080/...` 是因为本地跑；公网部署后该字段会是 https 公网 URL。
- 意义：证明**将要部署的这份产物**运行期 402 行为正确（与 SPEC01 本地结论一致），部署后 §5 的公网复测
  预期同样通过。

## 5. 阻塞项（需用户操作/授权，agent 不能自主完成）

| # | 阻塞步骤 | 为什么 agent 做不了 | 需要用户提供/执行 |
|---|---|---|---|
| 1 | Render 账号登录（dashboard，建议 GitHub OAuth） | 交互式登录、用户账号；**已定用 Render 免费版**（不绑卡） | 本人登录 <https://dashboard.render.com> |
| 2 | 连仓库建服务（Blueprint/Web Service，Root Dir=`services/frankfurter-fx`）+ 面板填 `PAY_TO` env(secret) → Deploy | 需登录态 + 从你的 public 仓库构建 | 部署后把 `https://frankfurter-fx.onrender.com` 记为 `<HOST>` |
| 3 | 仓库归属 | ~~origin 为上游~~ → **已决**：服务代码进独立仓库（fork），证据进 `kiteai-builder`；本地提交已备好，fork 创建后即 push | 用户在网页 fork（唯一剩余动作），push 由 agent 经 SSH 完成 |
| 4 | kpass 公网真实付费 + 反向不结算（验收 7/8） | 依赖 (2) 的公网 URL；且是真实付费/上链 | 部署后授权执行（testnet pieUSD 可免费领水） |
| 5 | `service.yaml` → `status: testnet` + `base_url` | schema 要求 testnet 必须有真实 `base_url`；`maintainer.github` 需真实 owner | 给 `<HOST>` 与 GitHub owner 后，由我改 + 跑 `npm run validate` |
| 6 | 看板提交 | 用户账号网页操作 | 本人提交（方向 `x402-service` + 仓库 URL + Commit SHA + 验收材料） |
| 7 | Track B 主网真实结算 | 花真 USDC.e | 显式出资 + 授权；否则默认跳过 |

> 说明：(5) 之所以本轮**没有**把 `service.yaml` 改成 `status: testnet`，是因为 schema 规定 `testnet`/`live`
> 必须带真实 `base_url`；在没有真实公网地址前填占位会导致 `npm run validate` 失败或信息不实。故 manifest
> 暂留 `status: draft`，待 (2)(3) 完成后一次性更新并校验。

## 6. 复现命令

**一键复现（本轮已实测通过）**：`week1/dryrun/local_dryrun.sh` —— 自包含，从 `week1/frankfurter-fx/`
构建并跑完整 §3–§4，末尾用 `week1/dryrun/decode_payment_required.py` 解码断言。实跑结果：
`build+vet OK` → 交叉编译 `ELF x86-64 static` → `/healthz` 200 → 未付款 `/v1/latest` 402 →
`RESULT: ALL_PASS` → 退出后 `port 8080 free`（trap 自动清理）。

```bash
bash week1/dryrun/local_dryrun.sh      # 一键：build + 交叉编译 + dry-run + 解码断言
```

手动等价步骤（即脚本内部所做的事）：

```bash
cd services/frankfurter-fx
GO="$HOME/.g/go/bin/go"            # 非交互 shell 里 go 由 g 管理

# 构建验证
"$GO" build ./... && "$GO" vet ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "$GO" build -o /tmp/fx-linux .   # == Dockerfile 构建阶段
file /tmp/fx-linux                                                       # ELF x86-64 static

# 运行期 dry-run（testnet）
"$GO" build -o /tmp/fx-native .
set -a; . ./.env; set +a
/tmp/fx-native &                   # 监听 :8080
curl -i http://localhost:8080/healthz                 # 200
curl -i "http://localhost:8080/v1/latest?base=USD"    # 402 + Payment-Required
# base64 解码 Payment-Required 头，断言 network/extra.name/extra.version/amount
kill %1                            # 停止，确认 port 8080 free
```

## 7. 本轮产物清单

- **仓库内（新增）**：`services/frankfurter-fx/Dockerfile`、`.dockerignore`、`render.yaml`、`verify_public.sh`
  （本轮已移除 Fly 专用的 `fly.toml` / `deploy.sh`）
- **week1（拷贝 / 生成）**：`0917-SPEC02.md`、`evidence2.md`、`frankfurter-fx/{Dockerfile,.dockerignore,render.yaml,verify_public.sh}`、
  `dryrun/{local_dryrun.sh,decode_payment_required.py}`（一键复现脚本，已实测 `ALL_PASS`）
- **不入库 / 不拷贝**：`.env`、`.env.mainnet`、`payer.key`、任何 `*.key`、`.kite-passport/`、编译二进制

## 8. 安全 / 纪律核对

- 未 `git commit` / `git push`；未部署任何平台；未花费资金；未发起上链交易。
- 新增文件**不含密钥**；`render.yaml` 用 `PAY_TO: sync:false`，值在 Render 面板填、**不写真实地址、不入库**。
- 本文件内所有地址脱敏。
- `kite.go` / `main.go` 未改动（协议根基 + 已验证逻辑）。
- 验证用本地服务已停止，无遗留监听端口。

### 8.1 入库泄漏自检（实测，非断言）

`git add -n services/frankfurter-fx/` 干跑，**只会**暂存以下 12 个安全文件：

```
.dockerignore  .env.example  .gitignore  Dockerfile  README.md  go.mod
go.sum  kite.go  main.go  render.yaml  service.yaml  verify_public.sh
```

`git check-ignore` 确认以下敏感路径**均被忽略**（不会进版本库）：

```
services/frankfurter-fx/.env          -> ignored
services/frankfurter-fx/.env.mainnet  -> ignored
services/frankfurter-fx/frankfurter-fx (二进制) -> ignored
.kite-passport                         -> ignored
```

`git ls-files | grep -iE '\.env|\.key|passport|secret'` 仅命中 3 个 `.env.example`
（模板示例，无真实值）。`payer.key` 仍在 `/tmp/fxpayer/`，未拷贝、未跟踪。
> 结论：当前工作区**可安全 push**（CI 不查密钥，故此处人工核验为唯一防线）。

## 9. kpass 命令面（已对 1.10.1 `--help` 核实，去歧义）

本机 `kpass` = `/Users/huangtao/.local/bin/kpass`，`kpass 1.10.1`。`kpass --help` 实测要点：

- **语法是冒号形式**：`agent:register`、`agent:session create|status|use|execute|list|attach`。
  仓库 README 的全空格写法（`agent register`）错误；官方文档的 `agent:register` 正确。
- **无 `sandbox` 命令** —— SPEC01 计划里的 `kpass sandbox on` 在 1.10.1 已不存在，Phase 4 无需该步。
- 关键子命令与必填参数（用于 Phase 4，已写进 SPEC02 与 `verify_public.sh` 末尾）：
  ```
  login init --email <e>            login verify --login-id <id> --code <OTP>
  me            health            status            config show
  wallet address [--chain ...]    wallet balance
  faucet drop --recipient <addr> --token <symbol>          # 无需鉴权
  agent:register --type <type>
  agent:session create --max-amount-per-tx <amt> --ttl <dur>
                       [--max-total-amount <amt>] [--assets <csv>] [--task-summary <text>]   # 需审批
  agent:session status --request-id <id> [--wait] [--timeout <s>]
  agent:session use --session-id <id>
  agent:session execute --url <url> [--method POST] [--headers <json>] [--body <json>]
  ```
- `agent:session execute --url` 印证 SPEC01 结论：请求由**后端服务端抓取** `--url`，故 Phase 4 必须公网
  https `<HOST>`；localhost 不可达。

> 仅运行了本地 `--help`（只读、无网络/无鉴权/无花费）。登录、审批、`execute` 真实付费均需用户交互与授权，
> 本轮**未执行**，故 Phase 4/5 仍为阻塞。

## 10. 进展日志（2026-09-18 补充：平台/仓库决策落地）

用户决策（交互确认）：① 部署平台 = **Render 免费版**（不绑卡、不花钱；Fly 移除）；② `PAY_TO` 维持现状
存 `.env`，是否随仓库公开后议；③ **Track B 主网跳过**（不花真钱）；④ 提交邮箱沿用本机 git config；
⑤ kpass 登录邮箱已提供（**不在任何入库文件中**，仅运行时用）。

据此完成：
- **仓库双轨确定**：服务代码 → `BrucerTao/kite-x402-services`（fork，看板提交 + Render 构建源）；
  证据/笔记 → `BrucerTao/kiteai-builder`（已存在，远端仅 init README）。
- `service.yaml` `maintainer.github`：占位 `bruce` → **`BrucerTao`**（看板归属硬要求）；`services/README.md`
  行同步 `@BrucerTao`。
- Fly 专用 `fly.toml`/`deploy.sh` **移除**；Render 产物 `render.yaml` + `verify_public.sh` 就位（§2）。
- 根 `.gitignore` 增加 `.qoder/`（连同已有 `.kite-passport/`、`.env`），`git add -n` 全库干跑确认
  **无任何** `.env*`(非 example) / `*.key` / `.kite-passport` / `.qoder` 入暂存。
- `templates/typescript-express/package-lock.json` 无关 npm churn 已还原（`"peer": true` ×3）。
- **本地提交已创建**：`7a4304b feat(services): add frankfurter-fx x402 wrapper`（14 文件，+687 行，
  基于 upstream `893a275` fast-forward；作者 = 本机 git 身份）。待 fork 建好后 push。
- **SSH 认证已验证**：`ssh -T git@github.com` → `Hi BrucerTao!`（push 通道可用）。
- **kpass 现状**：`kpass me` → HTTP 401（旧会话已过期），Phase 4 前需 `kpass login init --email <邮箱>` 重登。
- **安全复核**：`payer.key` 不在 week1（fxpayer 仅 go.mod/go.sum/main.go）；kiteai-builder 无任何
  `*.key`；邮箱类 PII 仅存在于 git 提交元数据（用户选择的本机身份），工作区文件零命中。
- `kiteai-builder` 本地已加 `.gitignore`（排除 `.DS_Store`/`.env*`/`*.key`）并提交 week1 证据（**未 push**，
  是否公开发布由用户决定）。

剩余用户动作（fork 已建、代码已 push，见下）：**连 Render 部署** → agent 跑 `verify_public.sh <HOST>` →
Phase 4（需 kpass 重登 + 授权）。

**2026-09-18 续**：fork `BrucerTao/kite-x402-services` 已建；remote 已配置（origin=本人 fork SSH、
upstream=gokite-ai）；`7a4304b` 已 push（`893a275..7a4304b main -> main`，远端 HEAD 核对一致）。CI 在
fork 默认禁用（Actions total_count=0，不阻塞看板，可后续在 Actions 页一键启用）。⚠️ 发现并修正文档：
Render Blueprint 只认仓库根目录 `render.yaml` → 改为**手动 Web Service** 路径（SPEC02 Phase 2 已更新），
`services/frankfurter-fx/render.yaml` 保留为声明式参考。证据仓库 `kiteai-builder` 本地提交 `32eb7a7`
（未 push，发布与否由用户决定）。

## 11. Phase 2-3-6 完成（2026-09-18：Render 公网部署 + 复测 + manifest 转正）

**Phase 2（部署）**：用户在 Render 面板手动建 Web Service（仓库 `BrucerTao/kite-x402-services` @
`7a4304b`，Root Directory=`services/frankfurter-fx`，Runtime Docker，Free 档，Health Check `/healthz`，
5 个 env，`PAY_TO` 为 Secret）→ 构建部署 **success**。公网地址：`https://frankfurter-fx.onrender.com`。

**Phase 3（公网复测，`verify_public.sh` 实测输出）**：
```
GET /healthz                      -> HTTP 200
  {"asset":"pieUSD","network":"eip155:2368","ok":true,"price":"$0.001"}
GET /v1/latest?base=USD (未付款)  -> HTTP 402 + payment-required 头
解码断言 4/4 PASS：
  accepts[0].network == eip155:2368        PASS
  extra.name == pieUSD                     PASS
  extra.version == 1                       PASS
  amount == 1000000000000000               PASS
  RESULT: ALL_PASS
```
附加预检（同日）：POST `/v1/latest` → 402（所有方法均门禁）；`/` → 404（无代理泄漏）；
`/healthz` → 200（免费）；TLS verify=0（有效证书、无浏览器验证型隧道；raw curl 可达 =
Passport 服务端可抓取）。响应头 `server: cloudflare`（Render 前置 CDN，正常）。

**Phase 6（manifest 转正）**：`service.yaml` `status: draft → testnet` +
`base_url: https://frankfurter-fx.onrender.com`；`services/README.md` 行同步 `testnet`。
`npm run validate` → **exit 0（✓ 2 service manifest(s) valid）**。`example_request` 保持
`query: {base: USD}` 形式（schema 约定：与 `base_url` 拼接，无需改动）。

**下一步**：Phase 4 真实付费（需 kpass 重登，旧会话已 401；warm-up 后 execute）→ Phase 5 反向不结算
→ Phase 7 commit+push（`status: testnet` 变更）+ 看板提交。

## 12. Phase 4-5 完成（2026-09-18：公网真实付费 + 反向不结算 + 根因排查）

### 12.1 背景曲折：首次付费调用「假 200」

首次对公网发起付费调用（fxpayer，SDK 默认签名窗口）结果诡异：`STATUS=200` 但 `BODY={}`、
无 `PAYMENT-RESPONSE` 头、**payer 余额纹丝不动** —— 表面成功、实际未结算。逐层排查：

1. **debug 副本加 ErrorHandler**（`/tmp/fxdebug`，仅临时，正式服务代码零改动）复现并拿到真因：
   `settlement failed: transaction_failed`；gin 日志出现
   `Headers were already written. Wanted to override status code 200 with 402`。
2. **直连 facilitator**（`/tmp/fxdiag`）拿原始响应：
   `/verify` → `{"isValid":true,"payer":"0xD2B1…d348"}`；`/settle` →
   `{"success":false,"errorReason":"transaction_failed","transaction":""}`（**连交易哈希都没有** =
   facilitator 未广播，广播前模拟就 revert）。
3. **eth_call 链上模拟** revert，错误数据 `0xdf8e4372`。
4. **破译两把钥匙**（keccak 选择器计算）：
   - pieUSD 的转账入口是非标准签名布局
     `transferWithAuthorization(address,address,uint256,uint256,uint256,bytes32,bytes)` = `0xcf092995`
     （打包 65 字节签名，而非 EIP-3009 标准的 `(v,r,s)` 三参数 `0x927da105`；由 SPEC01 成功结算交易
     `0xa56ccf24…` 的 input 解码反推）。
   - `0xdf8e4372` = **`AuthorizationNotYetValid()`**。

### 12.2 根因：Kite 测试网链头滞后 > 10 分钟

- SDK 签名 `validAfter = now − 600s`（10 分钟回拨，防时钟偏差）
- 实测（同一时刻）：真实时间 `14:59:46 UTC`，链最新块（22041163）时间戳 `14:30:24 UTC`
  —— **链头落后 ~29 分钟**，且出块间隔 ~36 分钟（停滞性慢出块）
- 于是链上 `block.timestamp < validAfter` → `AuthorizationNotYetValid()` revert →
  facilitator 报 `transaction_failed`
- SPEC01（09-17 17:02）当时链头无此滞后，故成功 —— 差异完全在链基础设施状态，**与服务代码无关**

### 12.3 Workaround：fxpayer2（宽签名窗口）

`/tmp/fxpayer2`（源码已拷贝至 `week1/fxpayer2/`）：完整复刻 x402 客户端付费流程
（GET 402 → EIP-712 签 EIP-3009 → 带 `PAYMENT-SIGNATURE` 头重放），仅把 `validAfter`
回拨从 600s 扩到 **2 小时**。facilitator `/verify` 不限制 validAfter 多旧（只验签名与
validBefore 未过期），故链滞后 30 分钟内也能结算。

### 12.4 Phase 4 证据：公网真实付费调用（验收 7）

```
$ ./fxpayer2 "https://frankfurter-fx.onrender.com/v1/latest?base=USD"
PAYER=0xD2B1B5D1C45c6313d08e49186E5B4faF4732d348
TARGET=https://frankfurter-fx.onrender.com/v1/latest?base=USD
STATUS=200
ELAPSED=3.609s
BODY={"amount":1.0,"base":"USD","date":"2026-09-18","rates":{"AUD":1.4045,"BRL":5.1359,
"CAD":1.401,"CHF":0.82565,"CNY":6.6976,"CZK":21.238,"DKK":6.523,"EUR":0.8726,"GBP":0.74939,
"HKD":7.8449,"HUF":317.87,"IDR":17823,"ILS":3.0377,"INR":95.88,"ISK":121.64,"JPY":157.89,
"KRW":1388.1,"MXN":17.1776,"MYR":4.0805,"NOK":9.4324,"NZD":1.7511,"PHP":62.803,"PLN":3.8076,
"RON":4.594,"SEK":9.853,"SGD":1.2784,"THB":33.355,"TRY":48.785,"ZAR":16.2724}}
```

**结算交易（链上 RPC 实证，比浏览器截图更强）**：

```
tx: 0x186f8e2090d80f39b0b83883bf3351e0085368fd6a1ad4c08ca2a5f554b9336f
    block 22041165 @ 2026-09-18 15:03:16 UTC | relayer(facilitator): 0x12343e…c78b
    from(payer)  0xD2B1B5D1C45c6313d08e49186E5B4faF4732d348
    to(pay_to)   0x1a8374b0F0D1074849e0cA31589532C2ad2806d8   == PAY_TO ✓
    value        0.001 pieUSD（1000000000000000 / 1e18）
浏览器复核: https://testnet.kitescan.ai/tx/0x186f8e2090d80f39b0b83883bf3351e0085368fd6a1ad4c08ca2a5f554b9336f
```

（另两笔历史结算：`0x0dd783a3…` = 本轮本地 debug 服务验证；`0xa56ccf24…` = SPEC01 本地付费调用。
三笔均为 payer→PAY_TO 各 0.001，facilitator 中继器一致。）

**余额对账**：faucet 100.000 − 3×0.001 = payer **99.997** ✓；PAY_TO **0.003** ✓（RPC eth_call）。

### 12.5 Phase 5 证据：上游失败不结算（验收 8）

```
$ ./fxpayer2 "https://frankfurter-fx.onrender.com/v1/latest?base=NOTACURRENCY"
TARGET=.../v1/latest?base=NOTACURRENCY
STATUS=404
BODY={"message":"not found"}          ← Frankfurter 上游对未知币种的真实 404 响应
```

- 买方已签名授权（同一付费流程走到上游），上游返回 404（≥400）→ 中间件 **不调用 settle**
- 链上复核：`PAY_TO` 转账交易仍为 3 笔（无新增）；payer 仍 **99.997**、PAY_TO 仍 **0.003**
  —— **分毫不差，未扣费** ✓
- 结论：`verify → upstream → settle` 顺序正确、仅上游 <400 才结算（验收标准 2 正反两面都实证）

### 12.6 过程中发现的两个上游缺陷（对 bounty 有价值，服务代码未改动）

1. **x402 Go SDK（gin middleware × ReverseProxy）**：反向代理流式转发触发 gin `Flush()` →
   `WriteHeaderNow()` 把 200 提前刷给客户端；此后 settlement 失败时 402 状态码与
   `PAYMENT-RESPONSE` 头无法送达（客户端看到 200+`{}`），结算成功时 `PAYMENT-RESPONSE`
   头同样丢失。本次交易哈希改由链上 RPC 取证。属 `coinbase/x402` SDK 的
   `responseCapture` 未实现 Flush 隔离所致，值得给上游提 issue。
2. **Kite 测试网出块停滞/滞后**（~36 min/块，链头时间戳落后墙钟 >10 分钟）触发
   `AuthorizationNotYetValid`，使 SDK 默认 600s validAfter 回拨的支付全部无法结算。
   kpass agent 等标准客户端同样会中招（其签名窗口同为 SDK 默认）。等待链恢复或由
   Kite 侧修复。

### 12.7 看板提交材料清单（用户网页操作）

- 方向：`x402-service`；仓库：`https://github.com/BrucerTao/kite-x402-services`（默认分支 main）
- Commit SHA：**`591749e`**（Phase 4/5 验证记录收口 commit；前序 `7a4304b` 服务接入、`98bbff1` manifest 转正）
- 公网地址：`https://frankfurter-fx.onrender.com`（`/healthz` 200；未付款 `/v1/latest` 402）
- `service.yaml`：`status: testnet`、`npm run validate` exit 0（✓ 2 manifests valid）
- 示例请求/响应：§12.4（200 + ECB 汇率 JSON）与 §12.5（404 + not found）
- 交易哈希：`0x186f8e2090d80f39b0b83883bf3351e0085368fd6a1ad4c08ca2a5f554b9336f`
  （testnet.kitescan.ai 可查，收款方 == PAY_TO）
