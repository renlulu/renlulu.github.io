---
title: "Prompt Caching 决定一切：Claude Code 的第一设计原则"
date: 2026-03-01 08:00:00 +0800
categories: [AI]
tags: [Claude Code, Prompt Caching, Agent, Anthropic]
---

Anthropic 的 Thariq Shihipar 写了一个系列叫 "Lessons from Building Claude Code"，[第一篇](https://x.com/trq212/status/2024574133011673516)讲的是 Prompt Caching。他自己的评价是：

> "One of the biggest realizations I've had working on Claude Code is that you fundamentally have to **design agents for prompt caching first**. Almost every feature touches on it somehow."

不是"可以优化"，是"必须围绕它设计"。Claude Code 团队对 cache 命中率设置了**监控告警**，命中率过低会触发 SEV（严重事故）。这不是锦上添花，是系统能跑起来的前提。

## 为什么 Prompt Caching 是生命线

长期运行的 Agent 产品和普通的单轮 API 调用不同。Claude Code 一个会话可能几十轮对话，上下文窗口不断膨胀。如果每轮都从头计算完整 prompt，成本和延迟会指数增长。

Prompt caching 允许**重用之前轮次的计算结果**。只要这一轮的 prompt 前缀和上一轮一样，前缀部分直接从缓存读取，只计算新增的部分。

这意味着：
- **成本大幅降低** — 缓存命中的 token 价格远低于重新计算
- **延迟大幅降低** — 不需要重新处理已经处理过的内容
- **更慷慨的 rate limit** — 成本低了，订阅用户能用得更多

反过来，如果 cache 命中率崩了，一切都会崩：成本飙升、响应变慢、用户体验暴跌。这就是为什么 Claude Code 团队把它当作和系统 uptime 一样的核心指标来监控。

## 核心原理：前缀匹配

Prompt caching 的匹配逻辑非常简单——**前缀匹配**。API 从请求的开头开始，逐字节比较，直到遇到 `cache_control` 断点。只要前缀和之前的请求完全一致，这部分就能命中缓存。

这带来一个直接的推论：**排列顺序决定一切。**

理想的 prompt 结构：

```
┌─────────────────────────────┐
│  静态系统提示 + 工具定义     │  ← 所有请求共享，缓存命中率最高
├─────────────────────────────┤
│  CLAUDE.md（项目配置）       │  ← 同项目内共享
├─────────────────────────────┤
│  会话上下文                  │  ← 同会话内共享
├─────────────────────────────┤
│  对话消息                    │  ← 最动态，放最后
└─────────────────────────────┘
```

越静态的内容越靠前，越动态的内容越靠后。这样，即使每轮对话消息不同，前面的系统提示和工具定义依然能命中缓存。

## 六种破坏缓存的方式（以及正确做法）

### 1. 在系统提示里嵌入动态信息

**错误做法：** 系统提示里写 `"当前时间是 2026-03-01 08:30:42"`。

每秒钟时间戳都变，意味着系统提示每秒都在变，缓存前缀从第一行就失效了。

**正确做法：** 系统提示保持静态。日期、时间等动态信息通过**下一轮的消息**传递：

```
// 不要改系统提示
// 在消息中插入系统消息
{ role: "system", content: "现在是周三" }
```

这样系统提示的缓存前缀完全不受影响。

### 2. 中途切换模型

**直觉：** 简单任务切到 Haiku 更便宜？

**现实：** Prompt cache 是**模型特定的**。Opus 的缓存和 Haiku 的缓存完全独立。如果 Opus 对话已经积累了 100k tokens 的缓存，切到 Haiku 意味着：

- Opus 的 100k token 缓存**全部作废**
- Haiku 需要从头计算完整 prompt
- 实际成本**反而更高**

**正确做法：** 如果需要用不同模型，用**子代理**。主模型准备一个精简的"交接消息"给子代理，子代理在自己的上下文里工作，不干扰主模型的缓存。

### 3. 中途改工具集

> "Changing the tool set in the middle of a conversation is one of the most common ways people break prompt caching."

工具定义是 prompt 的一部分，排在系统提示后面。删一个工具、加一个工具、甚至只是**改变工具的排列顺序**，都会让整个缓存前缀从工具定义那一段开始失效——后面所有的对话消息缓存也跟着废了。

**正确做法：** 工具集在会话开始时确定，之后**不要动**。

### 4. 用切换工具集来实现模式切换

Claude Code 有"计划模式"——进入后 Claude 只思考不执行。最初的实现方式是：进入计划模式时移除执行类工具，退出时加回来。

问题显而易见：每次模式切换都会破坏缓存。

**正确做法：** 保持所有工具始终存在，用 `EnterPlanMode` 和 `ExitPlanMode` 作为工具本身。模式切换通过系统消息告知模型"你现在在计划模式，不要调用执行类工具"，而不是真的把工具拿走。

工具集没变 → 缓存前缀没变 → 缓存命中。

### 5. 工具太多怎么办

Claude Code 有约 20 个工具。如果未来要支持几百个 MCP 工具呢？全部放进工具定义会让 prompt 膨胀，但动态加载又会破坏缓存。

**解决方案：延迟加载（defer_loading）**

```
// 工具存根（轻量级，占用很少 token）
{
  name: "some_mcp_tool",
  description: "Does something useful",
  defer_loading: true  // 不包含完整 schema
}

// 模型通过 ToolSearch 工具发现可用工具
// 只有选中时才加载完整 schema
```

工具存根是静态的、排列顺序固定的 → 缓存前缀稳定。模型需要某个工具时通过 `ToolSearch` 搜索，按需加载完整定义。

这样即使支持几百个工具，缓存命中率也不受影响。

### 6. 上下文压缩不复用前缀

上下文窗口快满时需要**压缩（Compaction）**——把当前对话摘要化，然后在新上下文中继续。

**错误做法：** 压缩后用新的系统提示或工具集开始新会话。

**正确做法：** 新会话必须使用**完全相同的**系统提示、用户上下文和工具定义。这样压缩前后的缓存前缀是一致的，之前的缓存可以被复用。

具体实现：
1. 把完整对话发给模型生成摘要
2. 用**完全相同的参数**（系统提示、工具定义）开始新会话
3. 把摘要作为第一条消息传入
4. 预留 "compaction buffer" 确保有足够空间容纳摘要输出

## 一个统一的思维模型

总结下来，所有经验都指向一个原则：

> **缓存前缀是神圣不可侵犯的。任何会改变前缀的操作，都要找到不改变前缀的替代方案。**

| 需求 | 错误做法 | 正确做法 |
|------|----------|----------|
| 更新动态信息 | 改系统提示 | 用消息传递 |
| 切换模型 | 直接换 | 用子代理 |
| 模式切换 | 增删工具 | 用工具建模状态 |
| 支持大量工具 | 全部加载 | defer_loading + ToolSearch |
| 上下文满了 | 新建会话 | 复用前缀的 compaction |

每一行的本质都一样：**前缀不要变**。

## 对 Agent 开发者的启示

这篇文章表面在讲 prompt caching，实际在讲一个更深的道理：**Agent 系统的架构要围绕 API 的物理约束来设计，而不是围绕逻辑上的优雅。**

逻辑上，进入计划模式时移除执行工具很优雅——模式清晰，语义明确。但物理上，这会破坏缓存，导致成本和延迟飙升。

逻辑上，简单任务切到小模型很合理——省钱。但物理上，重建缓存的成本可能比省下来的还多。

Thariq 的总结：

> **"Claude Code was built around prompt caching from day one. If you're building an agent system, you should do the same."**

如果你在构建 Agent 系统，在画架构图之前，先搞清楚 prompt caching 的前缀匹配机制。然后让一切设计决策服从这个约束。

---

*本文基于 Thariq Shihipar ([@trq212](https://x.com/trq212)) 的推文线程 [Lessons from Building Claude Code: Prompt Caching Is Everything](https://x.com/trq212/status/2024574133011673516) 整理，全文可在 [Techtwitter](https://www.techtwitter.com/articles/lessons-from-building-claude-code-prompt-caching-is-everything) 阅读。系列第二篇 [Seeing like an Agent](/posts/claude-code-seeing-like-an-agent/) 讲的是工具设计。*
