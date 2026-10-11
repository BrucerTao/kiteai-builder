# week4-commit.md — `BrucerTao/kite-x402-services` 提交记录（Week 4）

- **仓库**：<https://github.com/BrucerTao/kite-x402-services>（fork 自 `gokite-ai/kite-x402-services`）
- **分支**：`main`（默认分支，看板只认默认分支的 Commit SHA）
- **基线**：`d2d22bc`（Week 3 收尾状态）
- **本周共 4 个 commit，`d2d22bc..d16a966`**：前 2 个已推送（远端 HEAD `ad7d885`），后 2 个在本地 `main`（`git status -sb` = `## main...origin/main [ahead 2]`），**待用户确认安全审查结果后 push**
- **看板候选 SHA = `d16a966`**（push 后即远端默认分支 HEAD）。**验收标准 1–5 现已全部满足**，看板可以提交，见「看板提交材料」
- **本周实质推进**：新增 `usgs-earthquakes` 服务 → 部署公网 `https://usgs-earthquakes.onrender.com` → 本地 + 公网各一笔**真实付费结算**（收款方 == `PAY_TO`）→ 公网反向请求 400 **无结算** → manifest 转 `status: testnet` → 三条环境教训写进 AGENTS/ARCHITECTURE
- **commit author 本轮起改为** `BrucerTao <35832412+BrucerTao@users.noreply.github.com>`（仅仓库级 `git config`，**不重写历史**；此前两个仓库都是 `bruce <7****847@qq.com>`，公开仓库里的隐私暴露点）
- **push 前做了人工安全审查**（CI 不检查密钥泄漏）：14 文件 / +2892 行全量扫描 —— 无私钥/助记词/keystore、无 `.env`/`node_modules`/`dist`/`*.key` 入库、lockfile 与模板逐字节相同且 139 个依赖全指向 `registry.npmjs.org`、无 install 钩子、所有 `0x` 地址均为链上公开值。**查出 2 处隐私内容并已脱敏**：`1011-SPEC.md` 里 3 处本机绝对路径 `/Users/<name>/…` → `~/money/kiteai-builder/…`，2 处 QQ 邮箱字面量 → 打码；脱敏 amend 进 docs commit（原 `68c1274` → `ad7d885`，amend 发生在 push 之前，未重写任何已公开历史）

| # | SHA | 时间 | 类型 | 一句话 | 状态 |
|---|---|---|---|---|---|
| 1 | `75ca225` | 20:32 | feat | 新增 `usgs-earthquakes` x402 包装服务（USGS 地震目录，TS 模板） | ✅ 已推送 |
| 2 | `ad7d885` | 20:33 | docs | 入库 Week 4 执行规格 `1011-SPEC.md` + 全链路证据 `evidence-usgs.md`（含 push 前脱敏） | ✅ 已推送 |
| 3 | `041694d` | 22:00 | feat | 公网部署完成 + 付费验证通过 → manifest 转 `status: testnet` + `base_url`，服务 README 记两笔结算 tx | 🔒 本地，待确认 push |
| 4 | `d16a966` | 22:00 | docs | 重写 `evidence-usgs.md`（完整付费链路 + 三栏余额审计），AGENTS/ARCHITECTURE §9 补 7 条已知问题 | 🔒 本地，待确认 push |

**第 3/4 个 commit 的 push 前安全审查（2026-10-11 22:00）**：7 文件 / +196 −105 行全量扫描
（`git diff -U0`）—— 新增行内**零**未脱敏 EIP-3009 签名、零长 base64 串、零私钥形状裸 64-hex；
出现的 `0x` 值只有两笔结算 tx 哈希（链上公开、正是验收标准 5 要交的证据）与公开地址；无本机
绝对路径、无邮箱字面量；`git ls-files` 复核仍只有 5 个 `.env.example`，无 `.env`/`*.key` 入库。
付款方地址 `0x010F…4584` 只出现在 `evidence-usgs.md`，属链上公开值（沿用上周 `evidence.md`
已公开 fxpayer3 地址的同一口径）。

---

## Commit 1 — `75ca225` feat(services): add usgs-earthquakes x402 wrapper

> 2026-10-11 20:32 (+0800) · 12 files changed, **+2432**

把 [USGS 地震目录（FDSN event API）](https://earthquake.usgs.gov/fdsnws/event/1/) 包装为 Kite
测试网上的 x402 付费服务：未付费 `/v1/*` 返回 402 + `PAYMENT-REQUIRED`，只在**上游状态码 < 400**
时才走 verify → upstream → settle。**服务 `src/` 与模板逐字节相同**（`diff -r
templates/typescript-express/src services/usgs-earthquakes/src` 无输出），改的全是配置与文档。

选 USGS 的理由：数据**公共领域**（无 key、无公开速率限制、条款允许再分发），而 Open-Meteo
免费档**仅限非商业用途** —— 这让 `usgs-earthquakes` 成为目录里唯一有主网（Track B）晋升资格的候选。

| 文件 | 行数 | 内容 |
|---|---|---|
| `services/usgs-earthquakes/package-lock.json` | +1938 | 依赖锁（`@x402/{core,evm,express}` ^2.25.0、express ^5.1.0、tsx、typescript） |
| `services/usgs-earthquakes/src/index.ts` | +102 | 服务主体，**与模板一字未改**：express + x402 付费中间件 + 反向代理；`/healthz` 免费，`/v1/*` 付费门禁；base-path 拼接规则（`:69`）要求 `UPSTREAM_URL` 自带路径前缀 |
| `services/usgs-earthquakes/src/kite.ts` | +81 | 链常量与协议根基，**与模板一字未改**：testnet `eip155:2368`（pieUSD，18 位精度，EIP-712 `pieUSD`/`1`）与 mainnet `eip155:2366`（USDC.e，6 位精度）、金额解析、facilitator `https://facilitator.pieverse.io/v2` |
| `services/usgs-earthquakes/README.md` | +83 | 服务文档：端点/定价表、部署步骤、`### Do not pass jsonerror=true` 收费陷阱警告、"Verification status (2026-10-11)" 9 行表（含 2 项 pending）、USGS 数据署名 |
| `services/usgs-earthquakes/verify_public.sh` | +100 | 公网免费复测脚本：warm 重试 → `/healthz` 200 → 未付费 402 头 base64 解码断言。**比上周多第 5 条断言**（`resource.description` 含 `USGS`）—— 见「本轮暴露的问题」②，端口被上周遗留进程占用时只有这条能识破 |
| `services/usgs-earthquakes/service.yaml` | +37 | manifest：`schema: 1`、**`status: draft`**（付费验证未过，不写 `base_url`）、`network: eip155:2368`、`pay_to` 带引号、上游条款链接、单一端点 `GET /v1/query` + `example_request` + 3 条 `pitfalls` |
| `services/usgs-earthquakes/render.yaml` | +27 | Render 声明：native Node（非 Docker）、free 档、oregon、Build `npm ci && npm run build`、Start `node dist/index.js`、Health `/healthz`、`NODE_VERSION=22`、`PAY_TO: sync: false` |
| `services/usgs-earthquakes/package.json` | +25 | **只改 2 个字段**：`name: usgs-earthquakes`、`description`；scripts 与依赖同模板 |
| `services/usgs-earthquakes/.env.example` | +21 | 配置模板：`UPSTREAM_URL=https://earthquake.usgs.gov/fdsnws/event/1`（含 base path）、`KITE_NETWORK=testnet`、`PRICE_USD=0.001`、`PAY_TO` 占位 |
| `services/usgs-earthquakes/tsconfig.json` | +14 | 同模板（`npm run typecheck` 通过） |
| `services/usgs-earthquakes/.gitignore` | +3 | 排除 `node_modules`、`dist`、`.env` |
| `services/README.md` | +1 | 目录加行：`usgs-earthquakes | draft | eip155:2368 | GET /v1/query | @BrucerTao` |

## Commit 2 — `ad7d885` docs(repo): add week-4 spec and usgs-earthquakes evidence

> 2026-10-11 20:33 (+0800) · 2 files changed, **+460**

| 文件 | 行数 | 内容 |
|---|---|---|
| `1011-SPEC.md` | +344 | Week 4 执行规格（SPEC 04）：Context、**执行前实测事实**（上游/链/余额全部 2026-10-11 亲测，未沿用上周数字）、7 个决策点、**资金闸门**（含资金流表，证明推荐方案净变化为零）、Phase 0–6 共 36 步、9 行风险对策、纪律、14 项验证清单（现已勾 3 项）、3 条附带发现。状态头已改「部分执行（Phase 0–3 完成）」 |
| `evidence-usgs.md` | +116 | 全链路证据（week3 `evidence.md` 格式）：环境与参与者、§1 上游预检 4 项、§2 脚手架与本地 testnet 7 项、§3 未执行的 5 项验收（逐项写明阻塞原因）、§4 余额审计、6 条本轮暴露的环境/流程问题、免费复现命令、下一步用户动作 |

## Commit 3 — `041694d` feat(services): usgs-earthquakes live on public testnet

> 2026-10-11 22:00 (+0800) · 3 files changed, **+19 −13**

| 文件 | 内容 |
|---|---|
| `services/usgs-earthquakes/service.yaml` | `status: draft` → **`testnet`**，新增 `base_url: https://usgs-earthquakes.onrender.com`；`pay_to` 保持带引号 |
| `services/usgs-earthquakes/README.md` | 开头改成「`src/` 与模板逐字节相同，服务特定内容全在配置里」+ 新增 `Live:` 行；把带 pending 标记的 "Verification status" 表换成 9 行 **"Verified end-to-end (Kite testnet, 2026-10-11)"** 表，含两笔结算 tx、区块号、402 解码 5/5、反向不结算、轮末余额、`validate` 输出 |
| `services/README.md` | 目录行 `draft` → `testnet` |

## Commit 4 — `d16a966` docs(repo): record usgs-earthquakes evidence and testnet known issues

> 2026-10-11 22:00 (+0800) · 4 files changed, **+177 −92**

| 文件 | 内容 |
|---|---|
| `evidence-usgs.md` | 整篇重写：§2 本地补上真实付费一笔（200 / 7.813 s / tx `0xfbc83dfd…84a7` / block 22042203）；§3 公网 5 行（healthz、`verify_public.sh` ALL_PASS、付费 200 / 4.4 s / tx `0x9c66a3bd…b5ea` / block 22042202、反向 400 无 `PAYMENT-RESPONSE`）；§4 三栏余额审计（轮初 / 转入后 13:41:27Z / 两笔付费后 13:49:07Z）+ 上周沉没钱包单列；新增「本轮新发现」（按需出块的块间隔表 + 提交到上块耗时、EIP-3009 可重放教训） |
| `ARCHITECTURE.md` | §9 Known issues **补 7 条**：`usgs-earthquakes` 上线记录、Open-Meteo 免费档禁商用 → 不升 `live`、私钥别放 `/tmp`、EIP-3009 窗口内可重放、测试网按需出块、`.env` 空格截断、本地起服务前清端口；§10 文件索引补 3 个服务条目 + `evidence-usgs.md` |
| `AGENTS.md` | 「已知问题」同步上述 7 条的精简版（含 `usgs-earthquakes` 上线与两笔 tx） |
| `1011-SPEC.md` | 状态头改「**已执行完成（2026-10-11）**」并写入两笔 tx 与公网地址；14 项验证清单全部勾选，仅本地反向一项诚实标 `[~]`（两笔付费用尽余额后，缺钱的付款方会被 facilitator 以 `insufficient_funds` 挡在 402、到不了上游，无法证明 400-不结算路径 → 改由公网反向覆盖） |

---

## 本轮进度（全部完成）

| Phase | 内容 | 状态 |
|---|---|---|
| 0 | 执行前实测（上游 4 项预检、链头、三方余额） | ✅ 通过 |
| 1 | 脚手架（拷模板 → 配置 → `typecheck` → `diff -r` 空） | ✅ 通过 |
| 2 | 本地 testnet 免费部分（healthz 200、未付费 402 解码 5/5、base-path 拼接断言） | ✅ 通过 |
| 3 | 构建产物冒烟（= Render 上的 Build/Start 命令，`:8092` 复测）+ `npm run validate` | ✅ 通过 |
| 4 | 公网部署（push → Render → `verify_public.sh` ALL_PASS） | ✅ `https://usgs-earthquakes.onrender.com` |
| 5 | 真实付费（公网 1 笔 + 本地 1 笔）+ 反向不结算 | ✅ 两笔 tx 均 `status: 0x1`，收款方 == `PAY_TO` |
| 6 | manifest 转 `testnet`、AGENTS/ARCHITECTURE §9、看板提交 | ✅ 代码侧完成（`041694d` + `d16a966`）；看板提交待 push |

**资金闸门：已解除并已按预算花完（净变化为零）**。用户 2026-10-11 从收款方钱包 MetaMask 转入
**0.002 pieUSD**（13:41:27Z RPC 实测 `2000000000000000` units）；两笔付费各 0.001 回流到
`PAY_TO`，13:49:07Z 实测付款方 **0.0**、收款方 **0.002** —— 与 SPEC 里的资金流表逐位对上，
**没有重演上周 0.001 pieUSD 沉没的情况**。

闸门当初存在的原因：测试网水龙头两条渠道全停摆（`kpass faucet` prod/staging 均 "faucet is
disabled"；网页版 `/api/sendToken` 对所有 token 组合 500），`kpass agent:session execute`
又只放行 Kite 服务目录内的 host（自建公网 host 报 `host_not_in_executable_catalog`），上周
付款方 `0xB61D…5e5f` 的 0.001 pieUSD 因私钥放在 `/tmp` 被清理而**沉没**。

管路本身在缺钱时就已证明可用：fxpayer4 正确签出 EIP-3009 授权（`validAfter` 回拨 7200 s、
`value=1000000000000000`、`to=0x1a83…06d8`），带 `PAYMENT-SIGNATURE` 重试后服务返回 **402 +
`error: "insufficient_funds"`**，1.354 s，**未触达上游、未产生任何扣费**。

---

## 验收映射（方向 01 `x402-service`）

| 验收标准 | 证据 | 状态 |
|---|---|---|
| 1. 未付费 402 + `PAYMENT-REQUIRED`，`accepts[0].network` 正确 | 本地 `:8091`（`tsx`）、本地 `:8092`（`node dist/index.js`）、**公网** 三处都跑，base64 解码 **5/5 PASS**：`eip155:2368` / `extra.name=pieUSD` / `extra.version=1` / `amount=1000000000000000` / `resource.description` = `USGS earthquake catalog queries behind x402 on Kite`。公网由 `verify_public.sh` → `ALL_PASS`。`week4/evidence-raw/03`–`06`、`09`、`12` | ✅ 本地 + 公网 |
| 2. verify → upstream → settle，仅上游 <400 结算 | **公网反向实证**：`starttime=not-a-date&limit=5` → 上游 **400**（2.067 s），响应**无 `PAYMENT-RESPONSE` 头**、链上**无结算交易**、余额零变动。正向对照：同一端点合法参数 → 200 + 结算 tx。`week4/evidence-raw/15` | ✅ |
| 3. `example_request` 实测 2xx | manifest 的 `example_request` 经**付费**走服务 → **200**（公网 4.4 s / 本地 7.813 s），GeoJSON `FeatureCollection`、`metadata.status:200 api:2.8.1`、4 个 feature（M5.5–M5.9）。直连上游预检亦 200。`week4/evidence-raw/01`、`14`、`17` | ✅ |
| 4. `npm run validate` exit 0 | `✓ 3 service manifest(s) valid`（`d16a966` 工作区实测；两条 `shares pay_to` 是 note 非 error）。`week4/evidence-raw/10` | ✅ |
| 5. 本人真实付费调用记录（交易哈希） | **两笔**：公网 [`0x9c66a3bddafd5c0160cdd91e56f80603f8ff8360450b7222c2eb9f4f3145b5ea`](https://testnet.kitescan.ai/tx/0x9c66a3bddafd5c0160cdd91e56f80603f8ff8360450b7222c2eb9f4f3145b5ea)（block 22042202）、本地 [`0xfbc83dfd2ce2c4999ddbf204ad49a857d62d216e98c66fec9963d1ad76cf84a7`](https://testnet.kitescan.ai/tx/0xfbc83dfd2ce2c4999ddbf204ad49a857d62d216e98c66fec9963d1ad76cf84a7)（block 22042203）。两笔 receipt `status: 0x1`，标准 ERC-20 Transfer `payer → PAY_TO` 各 `0.001 pieUSD`。`week4/evidence-raw/14`、`16`、`16b`、`17`、`18` | ✅ |

交付材料：**齐**（公网 https 地址、`service.yaml`、校验结果、示例请求与响应、两个交易哈希）。

---

## 本轮暴露的环境/流程问题（均已写进 `evidence-usgs.md`）

1. **测试网链头停止推进（新发现，已定性）**：同一 head block `22042197`（`head_ts 2026-10-11T11:36:42Z`）
   在 12:01:40Z 与 12:24:25Z 两次采样间**没有变化**，滞后从 25.0 min 涨到 47.7 min，一度怀疑链停了。
   **付费后定性：测试网按需出块** —— 空闲时相邻区块间隔 7.4 / 63 / 16.5 / 12.9 分钟且每块恰好 1 笔
   交易，而两笔结算都在提交后 ~1 分钟内上块（22042202 / 22042203 相邻）。所以「head 时间戳很旧」
   不等于「结算会很慢」，不必等链追上墙钟；但 facilitator **提交即返回、不等确认**（HTTP 往返
   4.4 s / 7.8 s），receipt 必须自己轮询。`BACKDATE_SECONDS` 仍取 7200（够用）。
   已写进 ARCHITECTURE.md §9 与 AGENTS.md。
2. **本地端口被上周遗留进程占用，首次验证结果被污染（流程教训）**：一个 2026-10-04 17:08 启动的
   `node dist/index.js`（cwd `services/open-meteo-weather`，上周构建冒烟残留）连续 7 天占着 `:8080`。
   本轮第一次「本地验证」的请求全被它接走 —— `/healthz` 与 402 的 network/asset/amount 两个服务
   完全一样，看不出异常，只有 `resource.description` 是上周服务的兜底文案才露出马脚。已 kill、
   改用 `:8091`/`:8092` + **断言启动日志里的上游地址**重跑，`verify_public.sh` 新增第 5 条断言。
   **教训：本地验证前必须 `lsof -nP -iTCP:<port> -sTCP:LISTEN` 清场。**
3. **`.env` 里带空格的值必须加引号（真实 bug，两个服务都踩了）**：`.env` 靠 `set -a; . ./.env`
   由 shell 读取（应用本身不加载 dotenv），未加引号的 `SERVICE_DESCRIPTION=USGS earthquake …`
   会被截断、变量丢失，402 里只剩兜底文案。同样的 bug 在 `services/open-meteo-weather/.env:5`；
   `frankfurter-fx` 的写法（带引号）才是对的。**只影响本地开发**（Render 上 env 由面板注入）。
4. **付款方私钥放 `/tmp` 会丢**：上周 0.001 pieUSD 因此沉没 → 本轮改放仓库外 + `/tmp` 外的
   `~/.kite-x402/fxpayer4.key`（0600）。
5. **`jsonerror=true` 是收费陷阱**：USGS 加了这个参数后，坏查询返回 **HTTP 200** +
   `metadata.status:400`、`title:"Search Error"`。本 wrapper「上游 < 400 才结算」→ 这种失败查询
   **会被收费**。用**文档治理**（manifest `pitfalls` 第一条 + README 专节），不改代码：改代码会破坏
   「src 与模板逐字节相同」或触发「两模板必须同步改」的义务。
6. **kpass 门禁 / 水龙头停摆**：沿用 2026-10-04 实测结论，本轮复核仍然如此。

---

## 附带发现（影响已上线服务，非本轮代码任务）

- **Open-Meteo 免费档仅限非商业用途**（条款原文 "You may only use the free API services for
  non-commercial purposes"，定价表 Commercial use ❌）→ 已上线的 `open-meteo-weather`
  **必须保持 `status: testnet`，永不升 `live`**（付费调用即商业性转售）。
- **USGS 数据公共领域** → `usgs-earthquakes` 是目录里唯一适合做主网（`eip155:2366` / USDC.e）
  晋升与 Track B 的候选。
- **两个仓库的 commit author 都是 `bruce <7****847@qq.com>`**（公开仓库）→ 本轮起改 noreply，
  不重写历史（看板锚定 SHA，重写会作废）。

---

## 未入库内容（有意排除，非遗漏）

| 内容 | 原因 |
|---|---|
| `services/usgs-earthquakes/.env` | 仅含公开值（`PAY_TO` / `KITE_NETWORK` / `UPSTREAM_URL` / `PRICE_USD` / `SERVICE_DESCRIPTION` / `PORT`），纪律要求 `.env` 不入库；Render 面板以环境变量配置。已复制到 `week4/usgs-earthquakes/` |
| `~/.kite-x402/fxpayer4.key` | 付款方私钥（0600），**永不复制、永不打印、永不入任何仓库**（含本 builder 仓库）。已核验：`week4/` 下无 `.key`/`id_*`/`*.pem` 文件，且无任何与该私钥同值的字符串。文件里出现的 64-hex 串经逐条比对全是链上公开值（两笔 tx 哈希、EIP-3009 nonce、ERC-20/自定义事件的 `topic0`、log 里补零到 32 字节的地址与金额），**没有一个是私钥** |
| `/tmp/fxpayer4/` 下的 `fxpayer4`、`genkey` 二进制 | 编译产物；源码已同步到 `week4/fxpayer4/`，`go build` 可重建 |
| `services/usgs-earthquakes/dist/`、`node_modules/` | 构建产物与依赖（服务 `.gitignore` 已排除，`git check-ignore` 复核通过） |
| 原始证据（24 个文件） | 未入服务代码仓；已复制到 `week4/evidence-raw/`。其中 `14`/`15`/`17` 的 `signature` 字段已抹成 `0x<REDACTED>`；`/tmp/week4-public/` 下未脱敏的 `raw-paid.txt`、`raw-reverse.txt`、`raw-local-paid.txt` **有意不复制** |
| `templates/typescript-express/package-lock.json` 的 npm 无关改动 | 若有则提交前还原（本轮工作区 `git status` 干净，无此改动） |

**提交前 diff review（CI 不检查密钥泄漏，靠人工）**：分两轮做过。

- **第一轮（`75ca225` + `ad7d885`，push 前）**：14 文件 / +2892 行全量扫描。`git add -An` 干跑确认
  staged 面正好 12 个入库文件，无 `.env` / `node_modules` / `dist`；staged diff 内 64-hex（私钥形状）
  串**零命中**；文档里出现的 `0x…` 只有公开地址（收款方、两个付款方、pieUSD 合约）。查出 2 处隐私
  内容（本机绝对路径、QQ 邮箱）并已脱敏后 amend。
- **第二轮（`041694d` + `d16a966`，push 前）**：7 文件 / +196 −105 行。新增行内**零**未脱敏签名、
  零长 base64 串；`0x` 值只有两笔结算 tx 哈希与公开地址；无绝对路径、无邮箱字面量；`git ls-files`
  复核仍只有 5 个 `.env.example`。另外对 builder 仓 `week4/` 下**全部 26 个**唯一 64-hex 串与付款方
  私钥做了 **sha256 比对（私钥全程未打印、未读出）**，无一命中。

**关于 `week4/evidence-raw/07-local-harness-unfunded-smoke.txt`（签名已抹除）**：该文件原本含一笔
完整的 EIP-3009 授权 + 签名。取证时付款方余额为 0，facilitator 直接拒（`insufficient_funds`），
看起来是"废签名"；但**授权窗口 `validBefore=1791725095`（2026-10-11T13:24:55Z）在付款方到账后仍然
开着** —— 一笔"因缺钱而失败"的授权，在钱包被充值后就变成了**可重放的活凭证**（重放会把 0.001
pieUSD 从付款方转到 `PAY_TO`，两个都是自己的钱包，损失有限，但没必要留）。已把 `signature` 字段
替换为 `0x<REDACTED>` 并在文件头写明原因，其余保持原始 harness 输出不动。**该窗口已于
2026-10-11T13:24:55Z 过期，凭证现已彻底失效。**

同样处理也用在 Phase 5 的三份 harness 输出上：`week4/evidence-raw/14-public-paid.txt`、
`15-public-reverse.txt`、`17-local-paid.txt` 的 `signature` 字段一律抹成 `0x<REDACTED>`（`nonce`
保留 —— 它本来就写在结算交易的 log topic 里，是链上公开值，且没有签名就无法重放）。
`/tmp/week4-public/` 下的未脱敏原件 `raw-*.txt` **有意不复制**到 builder 仓。

**教训（值得写进 AGENTS/ARCHITECTURE）**：把付费 harness 的 `PAYLOAD`/`PAYMENT-SIGNATURE` 存进
证据文件时，要么当场抹掉签名，要么确认 `validBefore` 已过 —— EIP-3009 授权的可重放性只取决于
**窗口是否还开 + 付款方是否有余额**，与"上一次是否失败"无关。

## 中间产物同步说明

| 路径 | 内容 |
|---|---|
| `week4/usgs-earthquakes/` | 服务包：11 个入库文件（`.env.example`、`.gitignore`、`README.md`、`package.json`、`package-lock.json`、`render.yaml`、`service.yaml`、`tsconfig.json`、`verify_public.sh`、`src/index.ts`、`src/kite.ts`）+ 本地 `.env`（被 builder 仓 `.gitignore` 排除，只存在于工作区）。**与 `d16a966` 提交状态逐文件 `diff -q` 核验一致** |
| `week4/fxpayer4/` | 付款 harness 源码：`main.go`、`go.mod`、`go.sum`、`cmd/genkey/main.go`（不含私钥与二进制）。本轮从 `week3/fxpayer3/` 重建，并做了两处修改：私钥路径由硬编码 `/tmp/fxpayer3/payer.key` 改为 `PAYER_KEY_FILE` 环境变量 → 默认 `~/.kite-x402/fxpayer4.key`；`cmd/genkey` 的默认输出路径同步改掉 |
| `week4/evidence-raw/` | **24 个**原始取证文件。`01`–`10` 为免费部分（上游预检、本地 healthz+402、402 头/体、`PAYMENT-REQUIRED` base64、解码断言、harness 无余额冒烟、链状态、构建产物冒烟、`npm run validate`）；`11`–`18` 为 Phase 4/5 付费部分：`11` 公网 healthz、`12` `verify_public.sh` ALL_PASS、`13` 付费前三方余额、`14` **公网真实付费**、`15` **公网反向 400 无结算**、`16` 公网结算 receipt、`16b` receipt logs 解析、`17` **本地真实付费**、`18` 本地结算 + 轮末余额；另有 `healthz.json`、`local-healthz.json`、`local-server.log`、`receipt.json`、`receipt-local.json` |
| `week4/1011-SPEC.md`、`week4/evidence.md` | 与 `d16a966` 提交状态一致（`evidence.md` = 服务仓里的 `evidence-usgs.md`） |

免费复测入口（不花资金）：

```bash
curl https://usgs-earthquakes.onrender.com/healthz          # 免费档冷启动 ~50 s，先 warm
./week4/usgs-earthquakes/verify_public.sh https://usgs-earthquakes.onrender.com
```

---

## Render 实际部署配置（可复用于下一个服务）

Dashboard → New + → Web Service → 连 `BrucerTao/kite-x402-services`：

| 配置项 | 实际填写值 |
|---|---|
| **Root Directory** | `services/usgs-earthquakes` ← 最容易漏，不填会去仓库根找 `package.json` |
| Runtime / Plan / Region | Node（**非 Docker**）/ Free / Oregon |
| Build Command | `npm ci && npm run build` |
| Start Command | `node dist/index.js` |
| Health Check Path | `/healthz` |
| env | `NODE_VERSION=22`、`KITE_NETWORK=testnet`、`UPSTREAM_URL=https://earthquake.usgs.gov/fdsnws/event/1`、`PRICE_USD=0.001`、`SERVICE_DESCRIPTION=USGS earthquake catalog queries behind x402 on Kite` |
| env（Secret） | `PAY_TO=0x1a8374b0F0D1074849e0cA31589532C2ad2806d8`，勾 **Secret** |
| ⚠️ 不要设 | `PORT` —— Render 自己注入，写死会启动失败 |
| ⚠️ 不要漏 | `KITE_NETWORK` —— `src/index.ts:20` 的默认值是 **`mainnet`**，漏填不会报错，只会静默按主网 USDC.e（6 位精度）报价 |

得到的地址：<https://usgs-earthquakes.onrender.com>（免费档闲置 ~15 min 休眠）。

---

## 看板提交材料（push 后即可填）

| 字段 | 值 |
|---|---|
| 方向 | `x402-service`（方向 01） |
| 仓库 | <https://github.com/BrucerTao/kite-x402-services>（Public） |
| 本周 Commit SHA | `d16a966`（push 后 = 远端 `main` HEAD；4 个 commit 的区间 `d2d22bc..d16a966`） |
| 公网服务地址 | <https://usgs-earthquakes.onrender.com> |
| 端点 | `GET /v1/query`（$0.001 / 次，pieUSD on `eip155:2368`） |
| manifest | `services/usgs-earthquakes/service.yaml`，`status: testnet` + `base_url` |
| 校验结果 | `npm run validate` → exit 0，`✓ 3 service manifest(s) valid` |
| **真实付费 tx**（验收标准 5） | 公网 `0x9c66a3bddafd5c0160cdd91e56f80603f8ff8360450b7222c2eb9f4f3145b5ea`（block 22042202）；本地 `0xfbc83dfd2ce2c4999ddbf204ad49a857d62d216e98c66fec9963d1ad76cf84a7`（block 22042203）。两笔 receipt `status: 0x1`，Transfer `payer → PAY_TO` 各 0.001 pieUSD |
| 反向不结算（验收标准 2 反证） | `starttime=not-a-date` → 400，无 `PAYMENT-RESPONSE`，链上无新结算 |
| Changelog 摘要 | 新增 `usgs-earthquakes`（USGS 地震目录 FDSN event API 的 x402 包装，`src/` 与 TS 模板逐字节相同）→ 部署公网 testnet → 本地/公网各一笔真实付费 + 反向不结算全部通过 → manifest 转 `testnet`；AGENTS/ARCHITECTURE §9 补 7 条测试网运行教训 |

浏览器可查：把 tx 哈希拼到 `https://testnet.kitescan.ai/tx/<hash>`。

---

## 剩余动作（只有一项）

1. ⛔ **push `041694d` + `d16a966` 到 `origin/main`** —— 用户要求 push 前先看安全审查结果。
   第二轮审查已完成（见上文「提交前 diff review」），结论：**干净**。等一句确认即推。

已完成：转账（0.002 pieUSD，已按预算花完、净变化为零）、Render 部署、Phase 5 付费与反向验证、
Phase 6 文档与 manifest。

可选（不阻塞看板）：本 builder 仓的 `week4/` + `week4-commit.md` 提交 —— 该仓当前
`git config user.email` 仍是 `bruce <7****847@qq.com>`，推之前先切 noreply。
