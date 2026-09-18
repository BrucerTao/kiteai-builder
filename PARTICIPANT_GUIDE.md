# KiteAI 开发者活动（Bounty）参与与提交指南

欢迎参与 KiteAI 开发者激励活动！为了让大家顺畅完成提交并获得积分与奖励，请仔细阅读以下流程规范、方向指引及提交说明。

---

## 🧭 一、支持的研发方向（选择其一）

在开始创建项目前，请根据自己的技术栈与兴趣选择适合的方向进行开发：

| 方向名称 | 核心目标与说明 | 官方参考仓库 | 推荐技术栈 |
| :--- | :--- | :--- | :--- |
| **1. x402 付费服务**<br>`x402-service` | 将 HTTP API 包装为可发现、可调用的 x402 付费服务。要求未付费返回 402，按 `verify → upstream → settle` 顺序结算。 | [kite-x402-services](https://github.com/gokite-ai/kite-x402-services) | TypeScript / Express、Go / Gin 等 |
| **2. x402 Wrapper 模板**<br>`x402-wrapper` | 为其他开发语言补齐可运行、可测试的 Wrapper 模板，与官方模板保持相同的环境变量和 `/v1/*` 路由规范。 | [kite-x402-services](https://github.com/gokite-ai/kite-x402-services) | Python/FastAPI、Rust/Axum、Java/Spring Boot、Hono、Bun、Elixir 等 |
| **3. x402 服务准入评估**<br>`x402-evaluation` | 评估候选端点/服务的真实性、计费行为与稳定性，输出可复查的评估报告与巡检机制。 | [kite-x402-services](https://github.com/gokite-ai/kite-x402-services) | Node.js / TypeScript，定时任务、采集器与报告存储 |
| **4. Passport Skills Eval Runner**<br>`passport-skills-eval` | 将 Passport 行为用例转化为可执行、可评分的 CI Runner，完整执行用例并断言，可接入 GitHub Actions。 | [passport-skills](https://github.com/gokite-ai/passport-skills) | 依据官方现有测试格式，集成 CI 工作流 |
| **5. Passport DeFi 授权层**<br>`passport-defi-authorization` | 构建交易意图解码与授权策略校验工具（如 Uniswap、Morpho、Aerodrome 等主流 DeFi 协议适配与风控模拟）。 | [passport-skills](https://github.com/gokite-ai/passport-skills) | TypeScript，DeFi 协议适配器与策略校验引擎 |
| **6. 垂类 Seller Agent**<br>`seller-agent` | 打包可发现、报价、履约并结算的卖方 Agent（涵盖数据抓取、算力、内容生成、翻译等垂类）。 | [a2a-coordination-extension](https://github.com/gokite-ai/a2a-coordination-extension) | A2A 协议交互、各垂类执行框架 |

---

## 📌 二、提交与迭代机制

整个活动采用 **“单仓库持续演进”** 模式：

- **首次参与（第 1 周）**：提交 **GitHub 仓库链接** + **初始 Commit SHA**。系统会在内部审核通过后，自动为您向开源生态库（Electric Capital）发起收录申请。
- **后续周次（第 2 周及以后）**：**无需重新建立或更换仓库！** 必须在已登记的同一个仓库中继续做功能优化、重构、测试或维护，每周只需提交 **当周最新 Commit SHA**。

---

## 🛠 三、事前准备（必须完成）

在提交前，请务必完成：
1. **连接 Web3 钱包**：连接用于接收奖励的 EVM 钱包。
2. **完成 GitHub 账号绑定**：
   - 在看板点击完成 GitHub 授权绑定。
   - **重要**：你提交的仓库拥有者（Owner）必须是你绑定的 GitHub 账号，否则无法通过归属权校验。

---

## 📝 四、详细提交流程

### 1. 第 1 周（首次提交）
1. 在绑定的 GitHub 账号下创建并开源项目（必须为 **Public 公开仓库**）。
2. 选定上述 6 个方向之一，完成基础版本的开发并推送到 GitHub。
3. 打开 KiteAI Bounty 看板提交：
   - **选择贡献方向**（例如：`x402 付费服务` 或 `Passport Skills Eval Runner` 等）；
   - 填写 **GitHub 仓库 URL**（例如：`https://github.com/your-username/your-repo`）；
   - 填写 **Commit SHA**（如 `780fa43f`）；
   - 填写本次交付的功能说明与验收材料。
4. 提交后系统会进行 AI 预检查并进入管理员人工审核队列。

> **关于“已通过 · EC处理中”状态**：  
> 首次提交通过技术审核后，系统会自动向开源加密生态官方库（Electric Capital）发起 PR 登记。后台显示 `EC PR审核中 / EC处理中` 属于正常外部排队流程，**不影响** 您当周的打卡积分。

---

### 2. 后续周次（第 2 周及以后）
1. 继续在第 1 周登记的同一个 GitHub 仓库中开发：
   - 编写新特性、完善测试用例、修复 Issue、扩展接口或优化文档。
   - 推送代码到 GitHub 默认分支。
2. 打开看板进入新一周的提交入口：
   - 系统会自动锁定你绑定的历史仓库，**无需重复填写仓库地址**；
   - 仅需输入当周新增的 **Commit SHA** 以及对应的 **更新日志（Changelog）**；
3. 点击提交即可完成当周打卡。

---

## ⚠️ 五、注意事项与审核规范

1. **拒绝无意义空刷**：
   - 每周 Commit 必须包含实质性的代码推进、逻辑修改或功能迭代。
   - 严禁单纯修改标点符号、空格、空 Commit 等恶意刷取行为。系统包含 AI 代码审查与人工复核，违规者将被标记拒绝。
2. **仓库公开性**：
   - 仓库必须始终保持 **Public（公开）**，私有仓库将直接导致系统与开源库数据校验失败。
3. **Commit 归属一致性**：
   - 本地 Git 配置的 `user.email` / `user.name` 建议与 GitHub 绑定账号保持一致，确保 GitHub 网页端可正常关联至本人账号。
4. **审核反馈响应**：
   - 若状态变为“要求修改（`CHANGES_REQUESTED`）”，可查看管理员反馈意见，在仓库中补充修改推送后，重新提交新的 Commit 即可。
