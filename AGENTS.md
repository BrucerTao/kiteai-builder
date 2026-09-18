# AGENTS.md

本仓库是**付费 API 包装器的模板库 + 服务目录**，不是一个可直接运行的应用。根
`package.json` 只有 `validate`，没有 `start`；启动的永远是其中某一个 wrapper。

| 文档 | 内容 |
|---|---|
| [PARTICIPANT_GUIDE.md](PARTICIPANT_GUIDE.md) | 活动提交机制、事前准备、审核红线 |
| [ARCHITECTURE.md](ARCHITECTURE.md) | 组件关系、请求生命周期、启动步骤、不变量、已知问题 |
| [CONTRIBUTING.md](CONTRIBUTING.md) | 上游仓库的贡献流程、服务规则、状态机 |
| [README.md](README.md) | 项目总览、快速上手、Kite 网络参数、常见坑 |

`CONTRIBUTING.md` 讲的是给上游 `gokite-ai/kite-x402-services` 提 PR 的规则，
**不是**活动提交路径。

---

## 活动提交机制

目标是 Kite AI Bounty **方向 01 `x402-service`**：把 HTTP API 包装为可发现、
可调用的 x402 付费服务，未付费返回 402，按 `verify → upstream → settle` 结算。
后续每周仍是方向 01，在同一仓库持续推进。

**提交对象是自己账号下的 public 仓库 + 每周的 Commit SHA，不是给上游提 PR。**
上游 PR 只是可选加分项，不能替代看板提交。

### 事前准备（硬门槛）

1. 看板连接收款用 EVM 钱包
2. 看板完成 GitHub 授权绑定 —— **仓库 Owner 必须是绑定账号**，否则归属权校验失败
3. 本地 `git config user.email` 要能关联到绑定的 GitHub 账号

### 提交节奏

| 周次 | 做什么 | 看板填什么 |
|---|---|---|
| 第 1 周 | 建 public 仓库，完成基础版本，推到默认分支 | 方向、仓库 URL、初始 Commit SHA、功能说明与验收材料 |
| 第 2 周起 | 在**同一仓库**继续开发，推到默认分支 | 当周 Commit SHA + Changelog（仓库地址自动锁定） |

首次过审后系统会自动向 Electric Capital 发起收录 PR，看板显示 `EC处理中` 属
正常排队，不影响当周积分，此时不要动仓库。

### 审核红线

- 每周 commit 必须有实质代码推进；改标点、空格、空 commit 会被拒
- 仓库始终保持 Public
- `CHANGES_REQUESTED` → 按反馈修改推送 → 重新提交新 SHA

### 方向 01 验收标准

1. 未付费请求返回 402，带 `PAYMENT-REQUIRED` 头，`accepts[0].network` 是目标网络
2. verify → upstream → settle 顺序正确，只在上游状态码 < 400 时结算
3. 每个端点的 `example_request` 实测返回 2xx
4. `npm run validate` 退出码 0
5. 提交本人真实付费调用记录（交易哈希或 `kpass` 输出）

交付材料：公网 https 服务地址、`service.yaml`、校验结果、示例请求与响应、交易哈希。

---

## 环境

| 工具 | 状态 |
|---|---|
| Node / npm | v26.x（CI 用 22，本地更新无妨） |
| Go | 1.27.x，由 `g` 管理，`g ls` 查看。⚠️ 只有交互式 zsh 会自动加载 `~/.g/env`，脚本与非交互 shell 里要先 `. "$HOME/.g/env"` |
| `gh` | 未安装，建仓库与提 PR 走网页 |
| `kpass` | 未安装，付费测试必需：`curl -fsSL https://cli.gokite.ai/install.sh \| bash` |

### 测试网

- 水龙头：<https://faucet.gokite.ai>，或 `kpass faucet drop --recipient <地址> --token pieUSD`
- chain id `2368`、RPC `https://rpc-testnet.gokite.ai`、浏览器 <https://testnet.kitescan.ai>
- pieUSD `0x38129cf4CE5E183eFF248F42A7D345Bb1B47621A`，18 位精度

⚠️ 官方文档的命令是 `kpass agent:register` / `kpass agent:session create`（冒号），
仓库 README 写的是空格分隔。以 `kpass --help` 为准。

### 常用命令

```bash
npm install && npm run validate          # 仓库根：manifest 校验（CI 门禁）
cd services/<name> && npm run dev        # tsx watch；另有 typecheck / build
cd templates/go-gin && go build ./... && go vet ./...
```

CI 三个 job：manifest 校验（Node 22）、Go 模板 build+vet、所有含 `typecheck`
脚本的 TS 项目。Go 版本由 `templates/go-gin/go.mod` 钉住，别改那一行。

### 已验证的基线（不必重跑）

`npm run validate` 通过；两个 TS 项目 typecheck 通过；Go 模板 build+vet 通过；
`facilitator.pieverse.io/v2/supported` 实测含 `eip155:2368` 与 `eip155:2366`。

---

## 上游选型约束

**TS 模板要求上游真实端点落在根路径。** 代理剥掉 `/v1` 前缀后用
`new URL("/x", base)` 拼接，以 `/` 开头的相对引用会整体丢弃 base 的 path：

- `UPSTREAM_URL=https://api.open-meteo.com` + `/v1/forecast` → 打到 `/forecast` →
  404（真实端点是 `/v1/forecast`）
- 给 `UPSTREAM_URL` 加 `/v1` 后缀也没用，base path 一样被丢弃

**Go 模板不受此限制**（已实测）：`httputil.NewSingleHostReverseProxy` 的默认
director 会拼接 base path，`UPSTREAM_URL=.../v1` + `/v1/forecast` → 上游收到
`/v1/forecast`。也就是说**端点带前缀的上游（Open-Meteo、Frankfurter 等）只有走
Go 模板才配得通**。这个差异本身违反「两模板行为一致」不变量，属待修 bug。

选型还需复核上游条款是否允许代理/转售，结论写进 `upstream.terms_url`。带额度制
的上游不适合做付费服务：OpenAlex 已改为信用额度制，免费额度约 100 次/天，且其
terms 页返回 403。

---

## 硬不变量

完整版见 ARCHITECTURE.md 第 8 节。

- **只在成功时结算** —— verify → 上游 → settle 顺序不可调换（验收标准 2）
- **`kite.ts` / `kite.go` 保持原样** —— 链常量与 EIP-712 domain 是协议正确性的
  根基；`name`/`version` 与代币合约不一致时 facilitator 会拒绝每个签名，只报
  「settlement failed」，无线索
- **两个模板行为必须一致** —— 改一个要在同一 PR 改另一个
- **manifest 是封闭的** —— `additionalProperties: false`，未知字段直接校验失败
- `FACILITATOR_URL` 的 `/v2` 不能省；`pay_to` 在 YAML 里必须加引号（裸 `0x...`
  会被解析成整数）；`.env` 不入库，**CI 不检查密钥泄漏**，推送前自己 review diff

---

## 已知问题

- **示例服务 `open-meteo-weather` 上游路径不通**（status 仍 `draft`）：即上文 TS
  模板的根路径限制。详见 ARCHITECTURE.md 第 9 节
- **TS 与 Go 的 base path 拼接行为不一致**（已实测确认）：Go 保留 base path，TS
  丢弃。违反不变量，需在同一 PR 内修两边
- `services/open-meteo-weather/service.yaml` 的 `upstream.url` 写的是官网
  `https://open-meteo.com`，而 `.env.example` 是 `https://api.open-meteo.com`。
  schema 只校验 `^https://`，CI 不会报
- `templates/typescript-express/package-lock.json` 可能因 npm 版本差异产生
  `"peer": true` 被删的无关改动，提交前还原

---

## 协作约定

**自己的仓库（活动提交走这条）**

- Conventional Commits：`feat(services): add xxx`、`fix(template-ts): forward query string`
- 每周至少一个有实质内容的 commit，推到默认分支（看板只认默认分支的 SHA）

**给上游提 PR（可选）**

- 一个服务一个 PR。上游仓库无 open issue 且限制 issue 创建，任务来源只有
  CONTRIBUTING.md 与验收标准
- 服务目录名小写 `a-z0-9-`，且必须与 manifest 的 `name` 一致
- 状态机 `draft` → `testnet` → `live`；`base_url` 停摆会被退回 `draft`
- 新服务在 `services/README.md` 加一行；改模板要两个模板同步改

**两者都要守**

- 部署必须是公网 https：localhost、http、自签证书、浏览器验证型隧道都不行
  （Kite Passport 从其服务端抓取）

---

## 执行顺序

| # | 步骤 | 判据 |
|---|---|---|
| 1 | 看板绑定 GitHub + 连钱包，建 public 仓库并改 remote | `git remote -v` 的 origin 指向自己账号 |
| 2 | 装 kpass | `kpass --version` 有输出 |
| 3 | 拷模板到 `services/<name>/`，配 `.env`（`KITE_NETWORK=testnet`） | typecheck 或 `go build` 通过 |
| 4 | 本地验证 402 | base64 解码后 `network == "eip155:2368"`、`extra.name == "pieUSD"`、`extra.version == "1"`、`amount == "1000000000000000"` |
| 5 | 部署到公网 https | 外网 `/healthz` → 200；未付款 `/v1/...` → 402 |
| 6 | 水龙头领 pieUSD | `kpass wallet balance` 有余额 |
| 7 | 真实付费调用 | HTTP 200 + 上游响应体 + 结算交易哈希，且哈希在 testnet.kitescan.ai 可查、收款方是 `PAY_TO` |
| 8 | 反向验证失败不结算 | 打一个上游返 400 的请求，确认无结算交易 |
| 9 | 写 `service.yaml` + `README.md` | `npm run validate` exit 0；`status: testnet`；`example_request` 实测 2xx |
| 10 | 提交看板 | 方向 `x402-service`、仓库 URL、Commit SHA、验收材料 |

第 7 步是关键证据，其余步骤都为它服务。第 8 步是验收标准 2 的反向证明，容易漏。
