# kiteai-builder — Kite AI Bounty 方向 01 证据仓

本仓库是 **Kite AI Bounty 方向 01 `x402-service`** 的过程与证据仓库：存放每周的开发规格
（SPEC）、执行证据（evidence）、提交记录与笔记。**不放服务代码**——服务代码在同账号的
fork 仓库中开发、部署、提交看板。

## 外部资源

| 资源 | 说明 |
|---|---|
| Bounty 看板 | 每周提交入口：填方向、仓库 URL、当周 Commit SHA、验收材料。仓库 Owner 必须是绑定的 GitHub 账号，且仓库保持 Public |
| 上游模板库 [`gokite-ai/kite-x402-services`](https://github.com/gokite-ai/kite-x402-services) | 本仓库文档描述的主体：付费 API 包装器的模板库 + 服务目录（TS/Go 两个模板），活动提交就是 fork 它来做 |
| [`BrucerTao/kite-x402-services`](https://github.com/BrucerTao/kite-x402-services) | 本人 fork，**看板提交仓库**：frankfurter-fx 服务在这里开发与推送，看板认这里的 Commit SHA |
| `https://frankfurter-fx.onrender.com` | Week 1 上线的公网服务（Render 免费档，Docker）；`/healthz` 免费 200，未付款 `/v1/*` 返回 402 |
| Kite 测试网 | chain `eip155:2368`、RPC `https://rpc-testnet.gokite.ai`、浏览器 [testnet.kitescan.ai](https://testnet.kitescan.ai)、水龙头 [faucet.gokite.ai](https://faucet.gokite.ai)（pieUSD） |
| facilitator | `https://facilitator.pieverse.io/v2`（x402 verify/settle 服务） |

## 根目录文件

| 文件 | 含义 |
|---|---|
| README.md | 本文件：仓库定位、外部资源、目录说明 |
| AGENTS.md | AI 助手与新贡献者的入口速查：活动提交机制、方向 01 验收标准、本机工具链、上游选型约束、硬不变量、10 步执行顺序 |
| ARCHITECTURE.md | 上游模板库的架构说明：组件关系、请求生命周期（verify → upstream → settle）、8 条硬不变量、已知问题 |
| PARTICIPANT_GUIDE.md | Bounty 活动官方规范：6 个方向、单仓库持续演进机制、审核红线 |
| week1-commit.md | Week 1 在 fork 仓库的 3 个 commit 明细；**看板提交 SHA = `591749e`** |

## week1/ — 第 1 周工作目录

**目标**：把 Frankfurter（ECB 汇率，免 key）API 包装为 Kite 测试网上的 x402 付费服务，跑通
「本地全链路 → 公网 https 部署 → 真实付费调用 → 看板材料备齐」全流程，满足方向 01 的
5 条验收标准。

分两轮推进，每轮一份规格 + 一份精简证据：

| 轮次 | 规格 | 证据 | 内容 |
|---|---|---|---|
| 第 1 轮（09-17） | `0917-SPEC01.md` | `evidence.md` | 本地 testnet 全链路：402 门禁断言、自建付款方真实付费结算、上游 4xx 不结算、主网编码逻辑验证 |
| 第 2 轮（09-18） | `0917-SPEC02.md` | `evidence2.md` | 公网部署（Render）+ 外网复测、公网真实付费（结算 tx 链上实证）、反向不结算、manifest 转正 `status: testnet` |

配套目录：

| 目录 | 作用 |
|---|---|
| `frankfurter-fx/` | 商户服务源码快照（Go 模板），与 fork 仓库 `591749e` 提交状态一致 |
| `fxpayer/` | 自建付款方 harness：x402 客户端（GET 402 → EIP-3009 签名 → 重放 → 结算），SPEC01 本地付费用 |
| `fxpayer2/` | fxpayer 的宽签名窗口版（validAfter 回拨 600s → 2h），绕过测试网链头滞后 |
| `fxdiag/` | 直连 facilitator `/verify` `/settle` 的诊断工具，排查 settlement 失败用 |
| `dryrun/` | 本地一键复现：构建 + 交叉编译 + 运行 + 402 头解码断言 |

## 快速复现

```bash
bash week1/dryrun/local_dryrun.sh   # build + vet + 交叉编译 + 本地 dry-run + 402 断言
```

公网复测：`cd week1/frankfurter-fx && ./verify_public.sh https://frankfurter-fx.onrender.com`

## 安全约定

- `.env*`、`*.key`、`.kite-passport/`（agent JWT）、`.qoder/` 均不入库（见 `.gitignore`）
- 付款方私钥只存在于 `/tmp/fxpayer/payer.key`（0600），永不复制、永不上传
- 证据文件中的链上地址缩写显示；完整 `pay_to` 见 `week1/frankfurter-fx/service.yaml`（manifest 本身公开）
