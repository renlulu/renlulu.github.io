---
title: "拆解 Pi：一个 23k Star 开源 AI Agent 框架的架构设计"
date: 2026-03-15
categories: [AI]
tags: [pi, agent, architecture, openclaw, claude-code]
---

> Pi 是 Mario Zechner（libGDX 作者）做的开源 AI Agent 框架，23.7k stars。官网叫 `shittycodingagent.ai`——一个自嘲但很有底气的名字。本文从源码出发，拆解 Pi 的架构设计，看看构建一个 AI Agent 需要解决哪些核心问题。

## Pi 是什么

Pi 不只是一个 coding agent CLI，它是一整套 agent 构建工具包。核心是一个 monorepo，包含 7 个包：

| 包 | 职责 |
|---|---|
| `pi-ai` | 统一 LLM 通信层，支持 22+ 提供商 |
| `pi-agent-core` | Agent 运行时：循环、状态、工具执行 |
| `pi-coding-agent` | 旗舰产品：交互式终端编码 agent |
| `pi-tui` | 终端 UI 框架，差分渲染 |
| `pi-web-ui` | Web 组件，chat 界面 |
| `pi-mom` | Slack bot |
| `pi-pods` | vLLM GPU 部署管理 |

技术栈是 TypeScript（96.6%），用 TypeBox 做 schema，Biome 做 lint。

但让 Pi 真正有意思的不是这些功能列表，而是它的分层设计——这个分层已经被验证可以支撑完全不同形态的产品。

## 分层：Pi 架构的核心

Pi 最重要的架构决策是**三层分离**：

```
┌─────────────────────────────────────────────┐
│  产品层 (coding-agent)                       │
│  终端 CLI · 7 个工具 · Extension 系统 · TUI   │
├─────────────────────────────────────────────┤
│  Agent 运行时 (agent-core)                   │
│  Agent Loop · Steering · 并行工具执行 · 状态   │
├─────────────────────────────────────────────┤
│  LLM 通信层 (pi-ai)                         │
│  22+ 提供商 · EventStream · 跨提供商上下文转换  │
└─────────────────────────────────────────────┘
```

每层只知道自己该知道的事：

- **pi-ai** 只管和模型说话，不知道 agent 是什么
- **agent-core** 只管循环和状态，不知道工具具体干什么
- **coding-agent** 只管编码业务，不关心底层通信细节

这不是理论上的"好设计"。[OpenClaw](https://github.com/openclaw/openclaw)——一个支持 46 个消息渠道（WhatsApp、Telegram、Discord、Signal 等）的多渠道 AI 助手——就是直接基于 Pi 的这三层构建的：

```
Pi 自己的产品:     coding-agent → 终端编码助手
OpenClaw 基于 Pi:  Gateway → 46 渠道插件 → 个人 AI 助手
```

同一套 LLM 通信层和 Agent 运行时，长出了两个完全不同形态的产品。

## 第一层：pi-ai — LLM 通信的脏活

### 问题：每家 API 都不一样

用过多家 LLM API 的人都知道这个痛苦：

- OpenAI 的 tool call ID 可以有 450+ 字符带管道符，Anthropic 限制 64 字符且只允许 `[a-zA-Z0-9_-]`
- Anthropic 返回加密的 thinking signature，Google 有 thought signature，OpenAI 有 text signature——格式完全不同
- 有的模型支持 `reasoning_effort`，有的用 token budget，有的用枚举级别
- 每家上下文溢出的错误信息都不一样——Anthropic 说 "prompt is too long"，OpenAI 说 "exceeds the context window"，Google 说 "input token count exceeds the maximum"

Pi 用一个统一的接口抹平这些差异。

### 核心：EventStream

Pi 没用 RxJS、没用 Node Stream、没用回调，而是自建了一个极简的异步可迭代流：

```typescript
class EventStream<T, R> implements AsyncIterable<T> {
  push(event: T): void;          // 生产者推事件
  [Symbol.asyncIterator]();       // 消费者 for-await-of
  result(): Promise<R>;           // 等最终结果
}
```

支持背压、支持最终结果提取。LLM 的流式响应被标准化为 11 种细粒度事件：

```typescript
type AssistantMessageEvent =
  | { type: "text_delta"; delta: string; ... }
  | { type: "thinking_delta"; delta: string; ... }
  | { type: "toolcall_delta"; delta: string; ... }
  | { type: "done"; message: AssistantMessage }
  // ... 等等
```

注意 `toolcall_delta`——Pi 在流式传输过程中**实时解析不完整的 JSON**。工具参数不用等传完才能读取，流一半就已经 parse 出来了。

### transformMessages：最有工程价值的函数

这是 Pi 里最"脏"但最有价值的代码。当你在不同提供商之间切换模型时，对话历史里的消息需要转换：

```
OpenAI 的 tool call ID:  "call_xyz123|fc_abc456789..."  (450+ 字符)
                              ↓ transformMessages()
Anthropic 能接受的:       "call_xyz123_fc_abc456"  (64 字符以内)
```

它还处理：

1. **Thinking block 转换** — 加密的 thinking 只对同一模型有效，切换模型时转成普通文本
2. **孤立 tool call 修补** — 模型发出了工具调用但没返回结果？注入合成结果，防止下一轮 API 报错
3. **Thought signature 过滤** — Google 的 thought signature 是 base64 加密的，只在同一模型间有效
4. **错误/中断消息清理** — 跳过 `stopReason: "error"` 或 `"aborted"` 的消息

OpenClaw 利用这个能力实现了 auth profile failover——一家提供商挂了，自动切到另一家，对话上下文无缝传递。

### Thinking 级别统一

每家的"思考"实现都不一样，Pi 用一个 `ThinkingLevel` 统一：

| ThinkingLevel | Anthropic | OpenAI | Google |
|---|---|---|---|
| minimal | budget: 1024 | effort: "minimal" | LOW |
| medium | budget: 8192 | effort: "medium" | MEDIUM |
| high | budget: 16384 | effort: "high" | HIGH |
| xhigh | "max" (Opus 4.6) | — | — |

上层代码只需要说"我要 high 级别思考"，不需要关心底层怎么映射。

## 第二层：agent-core — Agent 运行时

### 问题：不只是 while 循环

大多数 agent 教程里的循环是这样的：

```
while (有 tool call) {
  执行工具 → 把结果喂回模型 → 看有没有新的 tool call
}
```

现实中这不够用。用户在 agent 执行过程中改主意了怎么办？多个工具可以并行的时候要不要并行？工具执行到一半出错了怎么恢复？

### 双层循环

Pi 的 agent loop 是两层嵌套的：

```
外层循环 (Follow-up)
  └── 内层循环 (Steering + Tool Execution)
        ├── 调 LLM → 发射 message events
        ├── 执行工具 (可并行)
        ├── 检查 Steering 消息 → 可中断剩余工具
        └── 注入新指令
```

**Steering**（转向）是关键创新。在工具执行过程中，外部可以往队列里塞消息：

```typescript
// Agent 正在执行一系列工具...
agent.steer({ role: "user", content: "停！不要部署。" });
// → 当前工具执行完后，跳过剩余工具，注入这条消息给 LLM
```

**Follow-up**（后续）处理另一种场景——agent 说完了，但有后续消息要注入：

```typescript
agent.followUp({ role: "user", content: "顺便也测试一下边界情况。" });
```

OpenClaw 用 steering 实现了：用户在 Telegram 发消息打断正在执行的 agent。

### 并行工具执行

Pi 默认并行执行工具，但做了精心设计：

1. **Preflight 串行** — 先逐个校验参数、跑 `beforeToolCall` 钩子（可以阻止执行）
2. **执行并发** — 通过校验的工具 `Promise.all` 并发执行
3. **结果按原始顺序返回** — 不管哪个先完成，返回顺序和 LLM 发出的顺序一致

为什么顺序重要？因为 LLM 看到的结果顺序会影响它的推理。如果模型先说"读文件 A"再说"读文件 B"，它期望先看到 A 的结果。

### 自定义消息类型

Pi 用 TypeScript 的声明合并（declaration merging）支持自定义消息类型：

```typescript
// 你的应用可以扩展消息类型
declare module "@mariozechner/pi-agent-core" {
  interface CustomAgentMessages {
    notification: { role: "notification"; text: string; timestamp: number };
  }
}
```

这些自定义消息存在 agent 状态里，但在发给 LLM 之前会被 `convertToLlm()` 过滤掉。上层应用可以在对话历史里插入自己的元数据，不会干扰 LLM。

## 第三层：coding-agent — 产品层

### 7 个内置工具

| 工具 | 做什么 | 细节 |
|---|---|---|
| read | 读文件 | 支持图片（自动缩放到 2000x2000）、offset/limit、截断到 4000 行 |
| bash | 执行命令 | 超时、流式输出、退出码追踪 |
| edit | 精确编辑 | find-and-replace，不是整文件覆盖 |
| write | 写文件 | 创建或覆盖，自动建父目录 |
| grep | 搜索内容 | 尊重 .gitignore，截断到 30KB |
| find | 搜索文件 | glob 模式，尊重 .gitignore |
| ls | 列目录 | POSIX 格式 |

### Operations 接口：工具与环境解耦

这是 Pi 工具设计里最聪明的部分。工具不直接调 `fs.readFile`，而是通过接口：

```typescript
interface ReadOperations {
  readFile(path: string): Promise<string>;
  stat(path: string): Promise<Stats>;
}

// 本地执行
const localOps = { readFile: fs.readFile, stat: fs.stat };
// SSH 远程
const sshOps = { readFile: sshClient.readFile, stat: sshClient.stat };
```

同一个 read 工具，本地跑、SSH 远程跑、容器里跑——零代码改动。Pi 的 ssh 扩展就是这么实现远程编码的，OpenClaw 用这个做工作空间沙箱隔离。

### Extension 系统

Pi 明确不用 MCP，用 TypeScript 原生扩展：

```typescript
export default function(pi: ExtensionAPI) {
  pi.registerTool({ name: "deploy", ... });       // 注册工具
  pi.on("tool_call", async (event) => { ... });     // 事件钩子
  pi.registerCommand("stats", { ... });              // CLI 命令
  pi.ui.setWidget("key", ["Line 1"]);                // UI 扩展
}
```

扩展可以做的事：

- 注册或替换工具
- 监听完整生命周期：`before_agent_start`、`tool_call`、`tool_result`、`compact`、`session_shutdown`...
- 注册 CLI 命令和快捷键
- 操作 UI（widget、overlay、dialog）
- 注册自定义 CLI flag（两遍参数解析：第一遍发现扩展，第二遍加载扩展注册的 flag）

OpenClaw 在 Pi 的 Extension 之上又建了自己的 Plugin SDK——用 Pi Extension 做工具级/钩子级扩展，用 OpenClaw Plugin SDK 做渠道级/产品级扩展。**两层扩展，各管各的抽象层级。**

### 系统提示设计

coding-agent 的系统提示是**动态构建**的：

```
基础指令（你是 pi，一个 coding agent harness）
+ 可用工具列表和使用指南
+ 项目上下文文件（AGENTS.md / CLAUDE.md，递归查找项目目录）
+ 已注册的 Skills
+ 当前日期和工作目录
```

可以被覆盖：

- `.pi/SYSTEM.md` — 项目级替换
- `APPEND_SYSTEM.md` — 追加而不替换
- `--system-prompt` — CLI 参数级

## TUI：终端渲染的工程细节

pi-tui 是一个独立的终端 UI 框架，有几个值得关注的技术点。

### 差分渲染

三种策略：

1. **首次渲染** — 全量输出，假定屏幕是干净的
2. **尺寸变化** — 清屏重绘
3. **增量更新** — 找到第一行和最后一行变化，只重绘变化区域

所有更新都用 **CSI 2026 同步输出协议**包裹——终端把一批输出作为原子操作渲染，零闪烁。这在 SSH 远程场景下尤其重要，减少了网络传输量。

### IME 支持

Pi 用一个自定义的 APC 序列（`\x1b_pi:c\x07`）作为 CURSOR_MARKER，精确告诉终端 IME 光标在哪里。这意味着中日韩输入法的候选窗口能正确定位——大多数终端 TUI 框架不处理这个。

## OpenClaw：Pi 架构的实战验证

OpenClaw 是理解 Pi 架构价值的最佳案例。它不是 fork Pi 然后魔改，而是**把 Pi 当引擎嵌入**：

```
用户消息
  ↓
[渠道插件] (Discord / Telegram / Slack / ...)
  ↓
[Gateway] (WebSocket 路由, 会话管理)
  ↓
[Pi Embedded Runner]
  ├── 解析 Model + Auth
  ├── 构建工具集 (Pi 的 codingTools + OpenClaw 自己的)
  ├── 创建 Session (Pi 的 SessionManager)
  ├── 流式调用 (Pi 的 streamSimple)
  └── 处理工具调用
  ↓
[回复分发] → 各渠道
```

OpenClaw 复用了 Pi 的什么：

| Pi 的能力 | OpenClaw 怎么用 |
|---|---|
| `streamSimple()` | 嵌入运行 agent 循环，流式结果通过 WebSocket 推到各渠道 |
| `SessionManager` | 直接复用做会话持久化，加了会话修剪和分组隔离 |
| `codingTools` | 选择性引入 read/write/edit，加上自己的 browser/canvas/voice |
| Extension 系统 | 自建 compaction-safeguard 和 context-pruning 扩展 |
| Model Registry | 包了一层 auth profile rotation + failover |

OpenClaw 加了什么 Pi 没有的：

- **46 个渠道插件**（Discord.js、grammY、Bolt 等）
- **DM 配对安全模型**（配对码、允许名单）
- **工具循环检测**（检测 agent 陷入死循环调用）
- **Auth profile 轮换**（一家挂了切另一家，带冷却时间）
- **上下文修剪策略**（按 cache TTL 窗口裁剪旧 turn）

## Pi vs Claude Code：两种哲学

读完 Pi 的源码，最明显的感受是它和 Claude Code 代表了两种不同的 agent 构建哲学：

| | Pi | Claude Code |
|---|---|---|
| 核心理念 | 最小内核 + 极致可扩展 | 全功能内置 |
| 扩展 | TypeScript Extension（全权限） | Hooks（shell 命令） |
| 工具协议 | 无 MCP（设计上排除） | 原生 MCP 支持 |
| 工具后端 | Operations 接口，可插拔 | 固定本地执行 |
| 运行模式 | CLI / Print / JSON / RPC | CLI |
| 权限系统 | 无内置（交给上层） | 内置权限审批 |
| 计划模式 | 无内置（可扩展实现） | 内置 Plan Mode |

Pi 的哲学是"不替你做决定"——不内置权限系统、不内置 plan mode、不内置后台 bash、不用 MCP。你需要什么，通过 Extension 自己搭。

Claude Code 的哲学是"开箱即用"——权限审批、子代理、MCP 集成、计划模式，全都内置好了。

两种都是有效的选择，取决于你是要一个**平台**还是要一个**产品**。

## 构建 Agent 的关键启示

从 Pi 的架构中可以提炼出几个通用的设计原则：

**1. 分层要真的分**

不是"代码放在不同文件夹"就叫分层。Pi 的分层是：每一层可以被单独替换，上层不知道下层的具体实现。OpenClaw 证明了这一点——它用 Pi 的下两层，但产品层完全不同。

**2. 流式传输需要专门设计**

LLM 的流式响应不是简单的字符串拼接。Pi 的 EventStream 处理了背压、取消、最终结果提取、不完整 JSON 解析。这些细节决定了用户体验。

**3. 跨提供商是工程问题，不是设计问题**

`transformMessages` 里没有什么精妙的算法，全是一个个 edge case 的处理——tool call ID 格式、thinking signature 兼容、孤立工具调用修补。但这正是框架的价值所在：把脏活集中到一个地方，让上层代码保持干净。

**4. 工具要和执行环境解耦**

Pi 的 Operations 接口是一个简单但有效的模式。工具定义"做什么"，Operations 定义"怎么做"。这让同一套工具可以在本地、SSH、容器、沙箱等不同环境运行。

**5. Agent loop 要支持中断**

现实世界的 agent 不是"启动后等它跑完"。用户会改主意，会发新消息，会想暂停。Pi 的 steering 队列是一个值得借鉴的模式——不是杀进程，而是优雅地注入指令。

---

*Pi 仓库：[github.com/badlogic/pi-mono](https://github.com/badlogic/pi-mono)，OpenClaw 仓库：[github.com/openclaw/openclaw](https://github.com/openclaw/openclaw)。*
