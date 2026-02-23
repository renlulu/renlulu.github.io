---
title: "Codex vs Crush vs OpenClaw：三个开源 Coding Agent 的架构抉择"
date: 2026-02-23 03:00:00 +0800
categories: [AI]
tags: [coding-agent, codex, crush, openclaw, architecture]
mermaid: true
---

> 同样是让 AI 写代码，OpenAI 用 Rust 重写了整个 CLI，Charm 团队内嵌了一个 POSIX Shell 解释器，OpenClaw 则把 Agent 接入了 30 个聊天平台。读完三个项目的源码后，我发现它们在几乎每个设计分叉点都做出了不同的选择——而这些选择背后是完全不同的工程哲学。

## 三个项目，三种定位

先说清楚它们各自在做什么：

| | Codex | Crush | OpenClaw |
|---|---|---|---|
| 厂商 | OpenAI | Charm（acqui-hire OpenCode） | 社区 + badlogic |
| 语言 | TypeScript → Rust | Go | TypeScript (Node.js) |
| LLM | 仅 OpenAI | 14+ provider | 15+ provider |
| 形态 | CLI + TUI + VS Code + Web | TUI | Gateway（30+ 渠道）+ CLI + Web |
| 平台 | macOS + Linux | 8 个 OS（含 Android、BSD） | 任何有 Node.js 的地方 |
| Stars | ~21K | ~20K | ~26K |
| 开源协议 | Apache 2.0 | FSL-1.1-MIT | AGPL-3.0 |

数字差不多，但产品哲学截然不同：

- **Codex**：我是 OpenAI 的官方工具，我只对接我们自己的 API，但我要做到**最安全、最快**
- **Crush**：我不绑定任何厂商，我要在**任何设备上用任何模型**写代码
- **OpenClaw**：写代码只是 Agent 能力之一，我要让它通过**任何渠道**服务任何人

这三种定位决定了后面所有的技术选择。

## 语言选型：为什么 Codex 要用 Rust 重写

三个项目的语言选择最直接地反映了它们的优先级。

**Codex** 在开源第 8 天（2025-04-24）就引入了 Rust。最初的 TypeScript CLI 用 Ink（React for CLI）构建，功能完整但有两个问题：Node.js 的启动延迟在 CLI 场景下很明显，更重要的是，沙箱需要和操作系统深度交互——macOS 的 Seatbelt、Linux 的 Landlock+seccomp、Windows 的 Job Object——这些用 TypeScript 做不了或做不好。到 2026 年 2 月，Rust 部分已经膨胀到 69 个 crate，TypeScript CLI 虽然保留但已不再是主力。

**Crush** 选 Go 是 Charm 团队的基因决定的。他们从 2019 年就在用 Go 做跨平台终端工具（Bubble Tea 30K stars），有成熟的 TUI 框架、跨平台编译能力、以及`mvdan.cc/sh` 这样的 POSIX Shell 解释器。Go 的交叉编译能力让他们能一次编译出 8 个 OS 的二进制文件——这对"任何设备"的目标至关重要。

**OpenClaw** 用 Node.js 是因为它本质上是一个**服务端应用**——Gateway 要长时间运行、处理 WebSocket 连接、对接 30+ 渠道 API。Node.js 的异步 I/O 模型天然适合这个场景。它还用了 Node.js 22 的实验性 `node:sqlite` 模块做记忆系统，不需要任何外部数据库。

## 命令执行：三种完全不同的安全模型

这是三个项目最核心的设计分歧。Coding agent 的杀手级能力是执行 shell 命令，但这也是最危险的能力——一条 `rm -rf /` 就能毁掉一切。三个项目用了三种截然不同的方式来平衡能力和安全。

### Codex：OS 级沙箱

Codex 的方法最"硬核"：让 agent 在真实 shell 里自由执行任何命令，但用**操作系统内核**限制它能访问的资源。

```
命令执行请求
    ↓
审批策略检查 (ApprovalPolicy)
    ↓
静态命令分析 (execpolicy crate)
  → 已知安全命令？跳过沙箱
  → 未知命令？进入沙箱
    ↓
OS 沙箱内执行
  macOS: Seatbelt sandbox profile
  Linux: Landlock + seccomp
  Windows: Job Object
```

三层防御：审批策略决定要不要问用户，`execpolicy` 的静态分析识别安全命令（`git status`、`ls` 等），沙箱限制进程可访问的文件路径和系统调用。

好处是 agent 在沙箱里可以做**任何事**，包括安装依赖、编译代码、运行测试——只是被限制在特定目录和网络范围内。坏处是必须为每个 OS 单独实现沙箱，这就是 Codex 只支持 macOS 和 Linux 的根本原因。

### Crush：内嵌 POSIX 解释器

Crush 的方法最出人意料：它根本不用系统的 bash。命令被送入一个 Go 实现的 POSIX Shell 解释器（`mvdan.cc/sh`），在 Go 进程内部执行：

```go
line, _ := syntax.NewParser().Parse(strings.NewReader(command), "")
runner, _ := s.newInterp(stdout, stderr)
runner.Run(ctx, line)  // 在 Go 进程内执行，不是 exec.Command
```

命令先被解析为 AST，然后在解释器的 `ExecHandler` 链中被拦截。`blockHandler` 看到的不是原始字符串，而是解析完管道、变量替换、转义之后的最终命令和参数。60+ 个命令被禁止（curl、wget、ssh、sudo 等），还有子命令级过滤——`npm install` 允许，`npm install --global` 禁止。

这个设计让 Crush 能在没有 bash 的环境（Windows、Android、BSD）上工作——因为它自带了一个。代价是内嵌解释器和真实 bash 存在行为差异，一些依赖 bashism 的命令可能不工作。

### OpenClaw：权限层级 + Docker 沙箱（可选）

OpenClaw 用最灵活但也最依赖配置的方式：

```
exec 工具调用
    ↓
工具策略管道 (tool-policy-pipeline)
  → 全局策略 → Agent 策略 → 群组策略 → 子 Agent 策略
    ↓
执行审批级别
  deny: 禁止一切命令
  safe-only: 仅白名单命令
  full: 允许所有
    ↓
可选 Docker 沙箱
  → 只读文件系统、无网络、无 Linux capabilities
```

在非交互场景（比如 Telegram Bot），通常配置 `security: "full"` 自动批准所有命令。安全靠 Hook 系统补偿——`before_tool_call` Hook 可以拦截危险命令。Docker 沙箱是可选的额外防护层。

### 对比总结

| | Codex | Crush | OpenClaw |
|---|---|---|---|
| 执行方式 | 真实 shell + OS 沙箱 | 内嵌 POSIX 解释器 | 真实 shell + 权限配置 |
| 安全边界 | 内核级 | AST 拦截 + 命令黑名单 | 策略配置 + Hook + Docker（可选） |
| Agent 自由度 | 沙箱内完全自由 | 黑名单外可执行 | 取决于配置 |
| 跨平台 | macOS + Linux | 8 个 OS | 取决于 Node.js |
| 安全深度 | 最深（OS 内核） | 中等（应用层） | 最灵活（可配置） |

**三种哲学的碰撞**：Codex 说"让你自由但关在笼子里"，Crush 说"告诉你什么不能做"，OpenClaw 说"你自己决定信任等级"。

## Agent 循环：殊途同归的核心引擎

三个项目的 agent 循环在宏观上惊人地相似——都是"LLM 思考 → 调用工具 → 返回结果 → 继续"。但实现细节差异很大。

### 调度架构

**Codex** 的循环由 `codex-core` 驱动，核心是 `run_turn()` 函数。它通过 SQ/EQ（Submission Queue / Event Queue）模式与所有前端通信——CLI、TUI、VS Code、Web 都用同一套 Op/Event 协议。这让 Codex 能在不改 core 的情况下扩展到任何界面。

**Crush** 的循环由 `SessionAgent.Run()` 驱动，通过 Fantasy 库的 `agent.Stream()` 进入工具循环。关键特点是**可变状态快照**——`csync.Value` 在循环开始前原子拷贝 tools、model、systemPrompt，运行中切换模型不影响当前请求。

**OpenClaw** 的循环由 `pi-embedded-runner` 编排。它不是自己实现循环，而是调用 `@mariozechner/pi-ai` 库的 `createAgentSession()`，然后通过 `subscribeEmbeddedPiSession()` 订阅每一步的结果。更上层有队列保证同一 session 的请求串行执行。

### 上下文管理

三个项目都面临同一个问题：对话太长时怎么办？

| | 触发条件 | 压缩方式 |
|---|---|---|
| Codex | token 达到上限 | `/compact` 命令 + 自动触发 |
| Crush | 剩余 < 20K（大窗口）或 < 20%（小窗口） | 自动 LLM 摘要替换早期历史 |
| OpenClaw | 超过上下文窗口的 80% | LLM 摘要，最多重试 3 次 |

Crush 的阈值策略最细腻——200K 窗口留 20K 绝对值缓冲，小窗口模型留 20% 相对值。这意味着它对不同模型有不同的触发时机。

三个项目都用 LLM 自身来生成摘要，而不是简单截断。但实现深度不同：OpenClaw 有完整的截断+摘要+重试链路，Crush 最简洁（触发后自动做一次），Codex 提供了手动 `/compact` 的选项给用户。

### 循环保护

| | 机制 | 实现 |
|---|---|---|
| Codex | token limit 触发 auto-compact | 隐式保护 |
| Crush | SHA-256 签名检测重复工具调用 | 窗口=10，最大重复=5 |
| OpenClaw | AbortController 用户中止 | 传播到 LLM 调用和工具执行 |

Crush 的循环检测最精巧——用工具名+输入+输出的 SHA-256 哈希来判断 agent 是否在做完全相同的事情。80 行代码解决了 agent "原地打转"的问题。

## LLM 集成：一个 Provider vs 多个 Provider

这是最直接反映产品策略的设计决策。

### Codex：深度优化单一 Provider

Codex 只支持 OpenAI，但做到了极致：
- **WebSocket-first**：不用 REST API，用 WebSocket 连接 OpenAI 的 Responses API。WebSocket 的 sticky routing 保证请求打到同一台 inference server，prompt cache 命中率最高
- **启动预热**：在用户输入第一条消息之前就预连接 WebSocket
- **Prompt caching**：工具列表排序、系统 prompt 结构化，最大化缓存命中

代价是用户**只能用 OpenAI 的模型**。如果你想用 Claude 或 Gemini，Codex 不是你的选择。

### Crush：多 Provider 抽象

Crush 通过 Fantasy 库统一了 14+ 个 provider 的接口。但源码暴露了一个关键现实——**抽象必然泄漏**。

最典型的案例：Anthropic 的 API 支持在 tool result 中携带图片，OpenAI 和 Google 不支持。Crush 的解决方案是悄悄重写对话历史——把图片从 tool result 抽出来，伪装成紧随其后的 user message。类似的 provider 专属处理散布在整个 codebase：Anthropic 需要 beta header 启用思考模式、OpenRouter 对特定模型追加 `:exacto` 后缀、`hyper` provider 根据模型名猜测该用哪个 SDK。

但 Crush 在 Anthropic provider 上实现了精细的三层 prompt caching（system message + 最后一个 tool 定义 + 最近 2 条消息），工具列表按名称排序防止缓存失效。这和 Codex 的缓存策略异曲同工——只是 Codex 用 WebSocket sticky routing 从传输层解决，Crush 用内容排序从应用层解决。

### OpenClaw：多 Provider + 熔断降级

OpenClaw 也支持 15+ provider，但它多了一套**生产级容错**：
- 多 key 轮换：同一 provider 配多个 API key，401/429 时自动切换
- 模型 fallback 链：`claude-opus-4-6 → gpt-4o → gemini-2.5-flash`，逐级降级
- Thinking 能力降级：`high → medium → low → off`
- 指数退避：失败后冷却 30s → 60s → 120s

这套机制对用户完全透明——工具循环不会因为某个 provider 临时不可用而中断。这是 OpenClaw 作为**服务端长期运行**的产品定位带来的需求——一个 CLI 工具可以让用户重试，但一个 7×24 运行的 Gateway 不能。

## Prompt 工程

三个项目的 system prompt 策略也反映了不同的产品理念。

| | 行数 | 风格 | Context Files |
|---|---|---|---|
| Codex | ~300 | 结构化指令 | AGENTS.md（自家格式） |
| Crush | 392 | XML 标签组织 11 个模块 | .cursorrules、CLAUDE.md、GEMINI.md、AGENTS.md **全部通吃** |
| OpenClaw | ~93 | 拼接式（身份+技能+记忆） | 自有记忆系统 |

Crush 的 prompt 策略最有意思：它不要求用户为自己写配置文件，而是**自动读取所有竞品的 context files**。这意味着已经在用 Claude Code 或 Cursor 的项目，切换到 Crush 时零迁移成本。

Codex 推广了 AGENTS.md 格式，Crush 读取了它，形成了一种有趣的生态寄生关系。

Crush 的 prompt 还有一个独特点：`git status --short` 和 `git log --oneline -n 3` 被自动注入 system prompt。Agent 在开始对话时就知道当前分支和最近 commit，不需要先调 git 工具。

OpenClaw 的 prompt 最短，但它有一个其他两个项目没有的东西——**记忆系统注入**。Agent 的 system prompt 里包含记忆搜索的指令，让 agent 在合适时机主动搜索历史信息。记忆用 SQLite + FTS5 + sqlite-vec 实现混合检索（70% 向量 + 30% 关键词），是三个项目中最完整的长期记忆方案。

## 扩展性：MCP、Skills、Hooks

三个项目都有扩展机制，但设计哲学完全不同。

**Codex** 是 MCP 的早期采用者。它实现了完整的 MCP client 和 server，让第三方工具通过标准协议接入。MCP 工具和内置工具在 agent 循环中一视同仁。Codex 还有一个自己的 Skills 系统。

**Crush** 同时支持 MCP 和 Agent Skills（`agentskills.io` 规范）。MCP instructions 在每次 `Run` 前注入，Skills 通过 `Discover()` 递归扫描项目目录发现 `SKILL.md` 文件。

**OpenClaw** 走了一条完全不同的路：**插件系统 + 20 个生命周期 Hook**。插件可以注册渠道、工具、HTTP 路由、WebSocket RPC 方法、命令。Hook 可以在 agent 执行的 20 个关键节点插入逻辑，支持 `block`（拦截）、`cancel`（静默取消）两种控制信号，按优先级排序执行。

这反映了三种扩展理念：
- Codex：**协议标准化**（MCP 让工具解耦）
- Crush：**规范兼容**（同时支持 MCP 和 Agent Skills，吃掉所有生态）
- OpenClaw：**平台化**（Hook 系统让第三方控制 agent 的执行流程）

## 文件编辑：细节见功夫

三个项目的文件编辑工具看起来做的是同一件事——修改代码文件。但安全保障差异很大。

**Codex** 用 `apply_patch` 工具，基于 Lark 语法定义的 patch 格式（不是正则）。文件写入在沙箱内自由进行，安全靠 OS 沙箱的路径限制。

**Crush** 的 edit 工具有三层保护：
1. **必须先读后改**：没有通过 `view` 工具读过的文件，拒绝编辑
2. **Stale read 检测**：编辑前检查 `modTime` 是否比上次 `view` 更新——如果用户手动改了文件，拒绝编辑
3. **SQLite 版本历史**：每次编辑的前后状态存入 SQLite，回滚粒度细到单次工具调用

**OpenClaw** 的 edit 工具来自 pi 库，是三个中最基础的——标准的搜索替换，没有额外的时间锁或版本历史。

Crush 的 modTime 检测是一个非常务实的设计——它解决了"agent 读文件后用户手动改了，agent 再写回去覆盖了用户的修改"这个 coding agent 的经典问题。

## 子 Agent

三个项目都支持子 agent，但粒度不同。

**Codex** 的子 agent 共享父 agent 的 MCP 连接但有独立的 TurnContext，通过 `multi_agents` 工具启动。

**Crush** 用 `fantasy.NewParallelAgentTool` 创建子 agent，支持**并行执行**——主 agent 可以同时启动多个子 agent。还有一个特殊的 `agentic_fetch` 子 agent，专门用 **small model**（而非 large model）分析网页内容，成本可能差 10-20 倍。

**OpenClaw** 的子 agent 通过 `sessions_spawn` 工具启动，每个子 agent 有独立的 session。更独特的是，OpenClaw 的子 agent 可以是**完全不同的 Agent 配置**——不同的模型、工具、工作区。

## 如果让我从零开始做一个 Coding Agent

读完三个项目的源码，我认为有几个经验是跨项目共通的：

### 安全不能后补

三个项目在第一天就考虑了安全。Codex 在初始 TypeScript 版本就有 Seatbelt 沙箱，Crush 从一开始就设计了命令黑名单和 Permission 系统，OpenClaw 的 exec 工具默认是 `deny` 的。安全机制是架构的一部分，不是事后加的功能。

### 上下文压缩是必需品

三个项目都实现了某种形式的上下文压缩——因为在真实的编码任务中，agent 经常需要读几十个文件、跑十几轮工具循环，对话长度很容易超过上下文窗口。这不是"nice to have"，是"must have"。

### Prompt caching 决定成本

Codex 用 WebSocket sticky routing，Crush 用工具列表排序 + 三层 cache control，两个项目都在想尽办法最大化 prompt cache 命中率。在 agent 场景下，工具循环的每一步都要发送完整的历史对话和工具定义——如果不缓存，成本是二次方增长的。

### 循环保护比你想象的重要

Crush 用 SHA-256 签名检测重复工具调用，OpenClaw 有用户中止机制，Codex 靠 auto-compact 隐式限制。Agent "卡住了反复做同一件事"是一个真实且常见的问题，需要在工程层面解决。

### 先读后改是基本纪律

Crush 强制要求先 view 再 edit，并且检测文件是否被外部修改。这解决了 LLM 幻觉一个文件路径然后向其写入垃圾的问题，也解决了和用户手动编辑冲突的问题。

### 选择你的取舍

Codex 选了安全深度（OS 沙箱），代价是平台广度（只支持 2 个 OS）。Crush 选了平台广度（8 个 OS），代价是安全深度（没有 OS 沙箱）。OpenClaw 选了渠道广度（30+ 渠道），代价是编码深度（工具系统最基础）。

没有哪个选择是"对"的——取决于你的目标用户是谁、在什么场景下使用。但**显式地做出取舍，并围绕它一以贯之地设计**，是三个项目共同的优秀品质。

---

*基于 Codex（2026-02-21，3913 commits，69 Rust crates）、Crush v0.44.0（2026-02-22，263 Go 文件）、OpenClaw v2026.2.18 源码分析。*
