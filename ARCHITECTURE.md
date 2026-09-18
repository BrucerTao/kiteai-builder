# 架构说明

本文档描述 `kite-x402-services` 的组件关系、启动步骤与使用方法。

面向读者的前置知识：了解 HTTP 与 REST API 即可，**不需要**区块链、智能合约或钱包开发经验。协议细节见 [x402](https://www.x402.org)，Kite 侧细节见 [README.md](README.md)。

---

## 1. 项目定位

这个仓库**不是一个可直接运行的应用**，而是一个「付费 API 包装器」的**模板库 + 服务目录**。

根 `package.json` 只有 `validate` 一个脚本，没有 `start`；仓库根目录不存在「启动整个项目」这回事。你启动的永远是其中**某一个** wrapper（模板的拷贝）。

它解决的问题：你已有一个 HTTP API，想让 Kite 网络上的 AI Agent 按次付费调用它。这里的模板给你套一层约 100 行的反向代理，它负责：

- 对 `/v1/*` 的未付款请求返回 `402 Payment Required` 并附带报价
- 通过 facilitator 校验买方签名
- 把已付款请求转发给你的上游 API
- 上游成功后才在 Kite 链上结算收款

仓库里**没有**智能合约、**没有**钱包代码、**不需要**付 gas。买方签一个 EIP-3009 代币授权，facilitator 负责广播交易并垫付 gas，稳定币直接进你的 `PAY_TO` 钱包。

> **与 `kite-services` 的区别**：Kite 官方也运营一个网关 `kite-services`，用它自己的钱包代理一批精选上游 API。本仓库是**自托管**的对应物 —— wrapper 由你部署，上游密钥由你持有，收款进你的钱包。

---

## 2. 组件关系图

### 2.1 运行时：谁调用谁

```
┌──────────────────┐
│  Kite Passport   │   买方 AI Agent（kpass CLI）
│  Agent           │   持有私钥，签 EIP-3009 transferWithAuthorization
└────────┬─────────┘
         │  ① GET /v1/xxx                → 402 + PAYMENT-REQUIRED（base64 报价）
         │  ② GET /v1/xxx                → 携带 PAYMENT-SIGNATURE 头重试
         ▼
┌───────────────────────────────────────────────────────┐
│   你的 wrapper（本仓库模板，部署在公网 https 上）        │
│                                                       │
│   ┌─────────────────────────────────────────────┐     │
│   │ kite.ts / kite.go    ← 唯一与 Kite 相关的代码 │     │
│   │  · 链常量：网络 ID、RPC、代币地址、精度        │     │
│   │  · MoneyParser：把 "$0.001" 换算成整数代币单位 │     │
│   │    并锁定 EIP-712 domain (name / version)    │     │
│   └─────────────────────────────────────────────┘     │
│   ┌─────────────────────────────────────────────┐     │
│   │ index.ts / main.go                          │     │
│   │  · 支付中间件：声明哪些路由收费、收多少        │     │
│   │  · 反向代理：剥掉 /v1 前缀，注入上游 API Key   │     │
│   └─────────────────────────────────────────────┘     │
└──────┬──────────────────────────────────┬─────────────┘
       │ ③ POST /verify                   │ ④ 转发请求（带上游凭证）
       │ ⑤ POST /settle                   │
       ▼                                  ▼
┌────────────────────────┐        ┌────────────────────┐
│  facilitator.          │        │   上游 API          │
│  pieverse.io/v2        │        │   (UPSTREAM_URL)   │
│  验签 + 广播 + 垫 gas   │        │   你自己的后端       │
└───────────┬────────────┘        └────────────────────┘
            │ ⑥ 链上结算
            ▼
     ┌──────────────────────────────────────────┐
     │  Kite 链                                  │
     │  eip155:2366 主网 → USDC.e  进 PAY_TO     │
     │  eip155:2368 测试网 → pieUSD              │
     └──────────────────────────────────────────┘
```

关键点：**买方永远拿不到你的上游 API Key**。密钥在第 ④ 步由代理注入（`index.ts:73`、`main.go:83-85`），且代理会主动删除请求里的 `PAYMENT-SIGNATURE` 头，不把支付信息泄漏给上游。

### 2.2 仓库内部：模板 → 拷贝 → 目录条目

```
        schema/service.schema.json          契约（封闭，不允许多余字段）
                    │
                    │ 被校验
                    ▼
templates/  ────复制────►  services/<name>/  ────►  scripts/validate.mjs
  │                            │                        │
  │  typescript-express/       │  service.yaml          │  npm run validate
  │  go-gin/                   │  README.md             │  （CI 门禁）
  │                            │  src/ 或 *.go          │
  │                            │  .env.example          ▼
  └── 两者行为必须完全一致 ──┘                     .github/workflows/ci.yml
```

| 路径 | 角色 |
|---|---|
| `templates/typescript-express/` | Node 22 + Express 5 + `@x402/express` 模板 |
| `templates/go-gin/` | Go 1.25 + Gin + `github.com/coinbase/x402/go` 模板 |
| `services/<name>/` | 模板的一份拷贝 + `service.yaml` + `README.md`，一个已部署服务一个目录 |
| `schema/service.schema.json` | manifest 的 JSON Schema（draft 2020-12），`additionalProperties: false` |
| `scripts/validate.mjs` | 校验所有 manifest，含 schema 表达不了的跨字段规则 |
| `services/README.md` | 服务目录总表，新增服务时要加一行 |

「配置而非分叉」是硬要求：`services/open-meteo-weather/src/` 与 `templates/typescript-express/src/` **逐字节相同**。新增一个服务，原则上只改 `.env` 与 `service.yaml`。

---

## 3. 一个 wrapper 的内部结构

每个 wrapper 只有两个源文件，职责严格分离：

### `kite.ts` / `kite.go` —— 链配置层（**不要改**）

| 导出 | 作用 |
|---|---|
| `KITE_MAINNET` / `KITE_TESTNET` | 两条链的完整常量：CAIP-2 网络 ID、RPC、代币地址、精度、EIP-712 domain |
| `FACILITATOR_URL` | `https://facilitator.pieverse.io/v2`，**`/v2` 不能省** |
| `kiteChainByName()` | 解析 `KITE_NETWORK` 环境变量，非法值直接抛错 |
| `kiteMoneyParser()` | 把 `"$0.001"` 这类美元价格换算成整数代币单位 |

`kiteMoneyParser` 是这一层存在的唯一理由：x402 SDK 只内置了固定几条链的默认稳定币，Kite 的两个稳定币不在表里。这个 parser 做三件事：

1. 只认自己那条链的 `network`，其它返回 `null` 交给下一个 parser
2. 用**十进制字符串的整数运算**换算单位，避开浮点误差（TS 版 `kite.ts:69-74`；Go 版用 `big.Float`）
3. 在 `extra` 里钉死 `name` / `version`，即 EIP-712 domain

第 3 点最容易踩坑：**EIP-712 的 name/version 必须与代币合约完全一致**，否则 facilitator 会拒绝每一个签名，表现为「settlement failed」而无更多线索。

### `index.ts` / `main.go` —— 应用层（按需改）

三段式，注释里已编号：

1. **构造 facilitator 客户端与 resource server**，注册链和 money parser（`index.ts:29-33`、`main.go:67-71`）
2. **声明收费路由**：一张 `路径模式 → 报价` 的表（`index.ts:43-60`、`main.go:52-64`）
3. **反向代理**：剥掉 `/v1` 前缀、注入上游凭证、剔除 hop-by-hop 头（`index.ts:66-95`、`main.go:75-106`）

只有当上游需要请求改写（换路径、加参数、转 body 格式）时才动第 3 段。

---

## 4. 请求生命周期

核心顺序是 **verify → 调上游 → settle**，且**只在上游返回状态码 < 400 时才结算**。

| 步 | 动作 | 失败时 |
|---|---|---|
| ① | Agent 请求 `GET /v1/forecast`，无签名头 | 中间件返回 `402` + base64 `PAYMENT-REQUIRED` 报价，请求不触达上游 |
| ② | Agent 用私钥签 EIP-3009 授权，带 `PAYMENT-SIGNATURE` 重试 | — |
| ③ | wrapper 调 facilitator `/verify` 校验签名 | 返回 `402`，不触达上游，不扣钱 |
| ④ | 校验通过，转发给上游 API | 上游不可达 → wrapper 返回 `502` |
| ⑤ | 上游返回 **< 400** → 调 `/settle` 链上结算 | 上游 4xx/5xx 或代理 502 → **跳过结算，买方不被扣款** |
| ⑥ | 返回上游响应体 + `PAYMENT-RESPONSE` 头（交易哈希） | — |

第 ⑤ 步的顺序是**不可协商的**。README 把它列为「Things that bite」的第一条：先扣款再验证是 Agent 对付费 API 的头号投诉。写自定义 wrapper 也必须保持这个顺序。

`502` 这个设计很巧：代理把「上游连不上」也映射成一个 ≥ 400 的状态码，于是自动复用了「不结算」这条规则，无需额外分支（见 `index.ts:85-86` 的注释）。

---

## 5. 启动步骤

### 5.1 前置条件

| 工具 | 版本 | 用途 |
|---|---|---|
| Node.js | 22+（CI 用 22） | TypeScript 模板、manifest 校验 |
| Go | 1.25 | Go 模板 |
| `kpass` CLI | — | 仅端到端付费测试需要，见 [Kite Passport 文档](https://docs.gokite.ai) |

### 5.2 仓库根目录：只做 manifest 校验

```bash
npm install
npm run validate
```

期望输出：

```
✓ 1 service manifest(s) valid
```

任何 manifest 有问题时退出码为 1，并逐条打印 `✗ ...`。CI 在 PR 和 push 到 `main` 时都会跑这一步。

`validate.mjs` 除 schema 之外还检查这些**跨字段规则**：

- `services/<dir>/service.yaml` 必须存在（每个目录都要有 manifest）
- manifest 的 `name` 必须等于目录名（`validate.mjs:44`）
- `services/<dir>/README.md` 必须存在（`validate.mjs:45`）
- 同一 manifest 内 `method + path` 不得重复（`validate.mjs:49`）
- `status` 不是 `draft` 时，每个 endpoint 必须有 `example_request`（`validate.mjs:51`）
- 多个服务共用同一 `pay_to` → 仅打印 note 警告，**不失败**（`validate.mjs:56`）

`pay_to` 有个专门的错误提示：YAML 里裸写 `0x...` 会被解析成整数，必须加引号。

### 5.3 启动一个 TypeScript 服务

```bash
cd services/open-meteo-weather
cp .env.example .env      # 然后编辑，至少填 PAY_TO
npm install
npm run dev               # tsx watch，改代码自动重启
```

其它脚本：`npm start`（不 watch）、`npm run build`（`tsc` 编译）、`npm run typecheck`（CI 会跑）。

`PAY_TO` 缺失会在启动时立刻抛错退出，不会静默跑起来（`index.ts:19`）。`UPSTREAM_URL` 缺失或非 http(s) 协议同样启动失败（`index.ts:22`）。

### 5.4 启动 Go 模板

```bash
cd templates/go-gin
cp .env.example .env
go run .
```

Go 版读取的环境变量、路由前缀、结算规则与 TS 版完全一致。启动时它会额外做一次 `SyncFacilitatorOnStart`（`main.go:99`），即开机就向 facilitator 拉一次 `/supported`，配置错误能立刻暴露而不是等到第一笔请求。

### 5.5 环境变量

两个模板读取同一组变量：

| 变量 | 必填 | 默认 | 说明 |
|---|---|---|---|
| `PAY_TO` | **是** | — | 收款的 Kite 钱包地址。缺失即启动失败 |
| `UPSTREAM_URL` | **是** | — | 被包装的上游 API 根地址 |
| `KITE_NETWORK` | 否 | `mainnet` | `mainnet`（USDC.e）或 `testnet`（pieUSD），其它值抛错 |
| `PRICE_USD` | 否 | `0.001` | 每次调用的美元价格。可带可不带 `$` 前缀，最多 6 位小数 |
| `UPSTREAM_AUTH_HEADER` | 否 | `Authorization` | 上游凭证的头名 |
| `UPSTREAM_AUTH_VALUE` | 否 | 空 | 上游凭证的值。**为空则不注入**，买方永远看不到它 |
| `SERVICE_DESCRIPTION` | 否 | `Paid API wrapped for the Kite network` | 出现在 402 报价的 `resource.description` 里 |
| `PORT` | 否 | `8080` | 监听端口 |
| `FACILITATOR_URL` | 否 | `https://facilitator.pieverse.io/v2` | 一般不用改，改了也**必须保留 `/v2`** |

`.env` 已在 `.gitignore` 中，**不要提交**，也不要在日志里打印剥离前的请求头。

### 5.6 启动成功的标志

```
kite x402 service on :8080 -> https://api.open-meteo.com/ (network eip155:2368, $0.001 per call to 0x1111...1111)
```

---

## 6. 使用方法

### 6.1 手工调用：观察 402

免费的健康检查：

```bash
curl -s localhost:8080/healthz
```

`KITE_NETWORK=testnet` 下：

```json
{"ok":true,"network":"eip155:2368","asset":"pieUSD","price":"$0.001"}
```

`KITE_NETWORK=mainnet`（默认）下 `network` 为 `eip155:2366`、`asset` 为 `USDC.e`。

未付款请求付费路由：

```bash
curl -i "localhost:8080/v1/forecast?latitude=52.52&longitude=13.41&current=temperature_2m"
```

得到 `HTTP/1.1 402 Payment Required` 和一个 base64 的 `PAYMENT-REQUIRED` 头。解开看报价：

```bash
curl -s -D - -o /dev/null "localhost:8080/v1/forecast?latitude=52.52&longitude=13.41" \
  | python3 -c "
import sys, base64, json
for line in sys.stdin:
    if line.lower().startswith('payment-required:'):
        print(json.dumps(json.loads(base64.b64decode(line.split(':',1)[1].strip())), indent=2))
"
```

测试网（pieUSD，18 位精度）下的实际输出：

```json
{
  "x402Version": 2,
  "error": "Payment required",
  "resource": {
    "url": "http://localhost:8080/v1/forecast?latitude=52.52&longitude=13.41",
    "description": "Paid API wrapped for the Kite network",
    "mimeType": "application/json"
  },
  "accepts": [
    {
      "scheme": "exact",
      "network": "eip155:2368",
      "amount": "1000000000000000",
      "asset": "0x38129cf4CE5E183eFF248F42A7D345Bb1B47621A",
      "payTo": "0x1111111111111111111111111111111111111111",
      "maxTimeoutSeconds": 60,
      "extra": { "name": "pieUSD", "version": "1" }
    }
  ]
}
```

注意 `amount` 是**整数代币单位**，不是美元。同样 `$0.001`：主网 USDC.e（6 位精度）是 `"1000"`，测试网 pieUSD（18 位精度）是 `"1000000000000000"`。这个换算正是 `kiteMoneyParser` 在做的事。

**路由契约只有两句话**：`/v1/` 下的所有路径都收费并代理（剥掉 `/v1` 前缀），`/healthz` 免费。x402 与 HTTP 动词无关，`/v1/*` 会保护所有 method —— 如果上游是只读的，应在 wrapper 里自行限制为 `GET`。

### 6.2 用 kpass Agent 端到端付费测试

**务必先在测试网跑通**：wrapper 用 `KITE_NETWORK=testnet` 部署，CLI 开 sandbox 模式，用免费的 pieUSD 付款。

```bash
kpass login init --email you@example.com
kpass login verify --login-id <上一步返回的 id> --code <邮箱里的 OTP>

kpass sandbox on                    # 切到测试网模式
kpass wallet address                # 你的 Agent 钱包地址
kpass faucet drop --recipient <该地址> --token pieUSD

kpass agent register --type service-tester
kpass agent session create \
  --task-summary "test my x402 wrapper" \
  --max-amount-per-tx 0.05 --max-total-amount 1 --assets pieUSD --ttl 1h
# 在浏览器打开打印出的批准链接，用 passkey 批准

kpass agent session execute --method GET \
  --url "https://your-host/v1/forecast?latitude=52.52&longitude=13.41&current=temperature_2m"
```

成功时会打印 HTTP 200、上游响应体和结算交易哈希。

验证主网路径：`kpass sandbox off`，对着 `KITE_NETWORK=mainnet` 的部署重跑，`--assets USDC`。

`--max-amount-per-tx` / `--max-total-amount` 是 Agent 侧的支出上限，用来兜住失控循环 —— 这也是「只在成功时结算」之外另一层买方保护。

### 6.3 网络参数参考

| | 主网 | 测试网 |
|---|---|---|
| CAIP-2 网络 ID | `eip155:2366` | `eip155:2368` |
| RPC | `https://rpc.gokite.ai` | `https://rpc-testnet.gokite.ai` |
| 结算资产 | USDC.e `0x7aB6f3ed87C42eF0aDb67Ed95090f8bF5240149e` | pieUSD `0x38129cf4CE5E183eFF248F42A7D345Bb1B47621A` |
| 代币精度 | 6 | 18 |
| EIP-712 domain | name `Bridged USDC (Kite AI)`, version `2` | name `pieUSD`, version `1` |
| Facilitator | `https://facilitator.pieverse.io/v2` | 同左 |

以上全部编码在 `kite.ts` / `kite.go` 中，正常使用无需改动。

---

## 7. 新增一个服务

完整清单见 [CONTRIBUTING.md](CONTRIBUTING.md)，流程概要：

1. **选一个值得付费的 API** —— Agent 按次付钱，所以它应该做模型离线做不到的事：实时数据、计算、动作。确认上游条款允许转售/代理，填进 `upstream.terms_url`
2. **从模板起步** —— 拷贝 `templates/typescript-express` 或 `templates/go-gin` 到 `services/<name>/`。`<name>` 为小写 `a-z0-9-`，且必须与 manifest 的 `name` 一致
3. **配置而非分叉** —— 只改 `.env`；仅在上游需要请求改写时动代理代码；`kite.ts` / `kite.go` 保持不变
4. **本地确认** 未付款请求返回 `402`，且 `accepts[0].network` 是你的目标网络
5. **部署到公网 https** —— Fly / Render / Cloud Run / VPS 均可。Kite Passport 从其服务端抓取你的 URL，因此 localhost、http、自签证书、浏览器验证型隧道**都不行**
6. **写 `service.yaml` 与 `README.md`**，在仓库根跑 `npm install && npm run validate`
7. **自己付一次款**，把输出贴进 PR
8. **在 `services/README.md` 加一行**

状态机：`draft` → `testnet` → `live`。

| status | 含义 | schema 强制 |
|---|---|---|
| `draft` | 只有代码，未部署 | — |
| `testnet` | 已部署，在 `eip155:2368` 收 pieUSD | 必须有 `base_url`，`network` 必须是 `eip155:2368` |
| `live` | 已部署，在 `eip155:2366` 收 USDC.e | 必须有 `base_url`，`network` 必须是 `eip155:2366` |

`base_url` 停摆时，维护者可能把服务退回 `draft`。

---

## 8. 必须保持的不变量

改代码前请确认没有破坏以下任何一条：

- **只在成功时结算** —— verify → 上游 → settle 的顺序不可调换
- **两个模板行为一致** —— 相同的环境变量、相同的路由前缀 `/v1/*`、相同的「成功才结算」规则。改一个必须在同一个 PR 里改另一个
- **`kite.ts` / `kite.go` 保持原样** —— 链常量和 EIP-712 domain 是协议正确性的根基
- **manifest 是封闭的** —— 未知字段会导致校验失败。需要新字段请先开 issue 讨论，不要就地加
- **价格用美元十进制字符串** —— 最多 6 位小数（USDC.e 精度所限）。目录里多数服务定价 $0.001–$0.05
- **每个已部署 endpoint 配一个 `example_request`** —— 且必须实测返回 2xx。买方从它起步，猜错要花真钱
- **仓库里没有密钥** —— `.env` 已被 git 忽略，但 CI **不检查**这一项，推送前自己 review diff

---

## 9. Known issues

### 示例服务 `open-meteo-weather` 的上游路径不通

代理会剥掉 `/v1` 前缀（`index.ts:67`、`main.go:104`），而 `.env.example` 里配的是 `UPSTREAM_URL=https://api.open-meteo.com`。于是 `/v1/forecast` 被转发到 `https://api.open-meteo.com/forecast` —— 但 Open-Meteo 的真实端点是 `/v1/forecast`：

```
https://api.open-meteo.com/forecast      → 404
https://api.open-meteo.com/v1/forecast   → 200
```

**且无法通过给 `UPSTREAM_URL` 加路径后缀来绕过**，因为 TS 模板用的是 `new URL("/forecast", base)`，以 `/` 开头的相对引用会整体替换掉 base 的 path：

```
UPSTREAM_URL=https://api.open-meteo.com/v1  →  仍然命中 /forecast（base 路径被丢弃）
```

影响面：

- 404 ≥ 400，所以**不会结算、不会扣买方的钱**，但示例服务事实上不可用
- Go 模板用的是 `httputil.NewSingleHostReverseProxy`，其默认 director **会**拼接 base path，因此 `UPSTREAM_URL=.../v1` 在 Go 下可能是正常的。若如此，两个模板行为不一致，违反第 8 节的不变量。（此项尚未在装有 Go 工具链的环境实测确认）

可选修法：给 TS 模板改成保留 base 路径的拼接方式；或让代理不剥前缀；或为需要路径前缀的上游增加一个显式配置项。

### `service.yaml` 与 `.env.example` 的上游地址不一致

`services/open-meteo-weather/service.yaml` 里 `upstream.url` 写的是 `https://open-meteo.com`（官网），而 `.env.example` 里是 `https://api.open-meteo.com`（API）。schema 只校验 `^https://`，不校验两者是否对应，所以 CI 不会报。

---

## 10. 文件索引

| 文件 | 说明 |
|---|---|
| `README.md` | 项目总览、快速上手、Kite 网络参考、常见坑 |
| `CONTRIBUTING.md` | 贡献流程、服务规则、状态生命周期、提交规范 |
| `schema/service.schema.json` | manifest 契约 |
| `scripts/validate.mjs` | manifest 校验器（`npm run validate`） |
| `templates/typescript-express/src/kite.ts` | Kite 链常量 + money parser（TS） |
| `templates/typescript-express/src/index.ts` | 支付中间件 + 反向代理（TS） |
| `templates/go-gin/kite.go` | Kite 链常量 + money parser（Go） |
| `templates/go-gin/main.go` | 支付中间件 + 反向代理（Go） |
| `services/open-meteo-weather/` | 示例服务（当前状态 `draft`） |
| `.github/workflows/ci.yml` | CI：校验 manifest、构建 Go 模板、typecheck 所有 TS 项目 |
