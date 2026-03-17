---
title: "为什么 AI Agent 总在反问你，而不是直接去做？"
date: 2026-03-17
categories: [AI]
tags: [ai-agent, prompt-engineering, claude-code, codex, system-prompt]
---

> 你问 AI Agent："记忆文件在哪？"
> 它回你："你想让我查一下吗？"
> 你说："查。"
> 它又问："要查哪个目录？"
> ……

这个体验令人抓狂。明明 Agent 手里有工具、有权限，却像一个过度谨慎的实习生，每一步都要请示。

这不是模型不够聪明的问题，是 **系统设计** 的问题。

## 问题的本质

把一个 LLM 接上工具，不等于你得到了一个好用的 Agent。大多数人搭 Agent 时会写这样的 system prompt：

```
你是一个 AI 助手。你有以下工具：
- exec：执行命令
- read_file：读取文件
- search：搜索文件内容
```

然后期望它能像 Claude Code 一样高效。但实际效果是——模型把工具列表当成了"菜单"，每次都问用户："你要点哪个？"

根本原因：*你告诉了它有什么，但没告诉它该怎么用。*

LLM 的默认行为模式是对话。它被训练成"先理解意图、再确认、最后执行"。这在普通聊天里是好品质（避免误操作），但在 Agent 场景里变成了摩擦。

## Claude Code 的解法

Claude Code 在这方面做得相当极致。拆解它的 system prompt，核心策略有四个：

### 1. 行为规则优先于能力描述

Claude Code 的 system prompt 不是先介绍"你有什么工具"，而是先定义行为准则：

- *先做再说，不要反问。*
- *用户问文件内容？先读了再回答，不要问"要我读吗"。*
- *尝试最简单的方法，不要绕圈子。*

这是关键的认知转变——system prompt 不是说明书，是**行为契约**。

具体来说，它会写类似这样的指令：

```
Go straight to the point. Try the simplest approach first.
Lead with the answer or action, not the reasoning.
Do not propose changes to code you haven't read.
If a user asks about a file, read it first.
```

注意最后一句："如果用户问到一个文件，先读它。" 不是"问用户要不要读"，是"直接读"。

### 2. 注入环境上下文

Claude Code 启动时，系统会自动注入大量环境信息：

```
- 工作目录: /Users/xiaohuo/workspace/project
- Git 分支: main
- 最近 commits: ...
- 平台: macOS
- 当前日期: 2026-03-17
```

这些信息让 Agent 不需要"探索"就知道自己在哪、在做什么。对比一下：

| 场景 | 没有上下文 | 有上下文 |
|------|----------|---------|
| "项目结构是什么？" | 先问目录 → 再 ls → 再回答 | 直接 ls 当前目录 → 回答 |
| "最近改了什么？" | 问用户 git repo 在哪 | 直接 git log → 回答 |
| "读一下配置文件" | 问"哪个配置文件？" | 先 glob 查找 → 读取 → 回答 |

少一轮交互，体验就好一个量级。

### 3. 场景→工具的映射

Claude Code 不只是列工具名，而是建立*场景到工具的映射*：

```
- 读文件用 Read，不要用 cat
- 搜索文件用 Glob，不要用 find
- 搜索内容用 Grep，不要用 grep
- 简单搜索直接用 Glob/Grep
- 复杂探索才用 Agent 子任务
```

这比"你有 Read、Glob、Grep 工具"有用得多。模型不需要每次都"思考"该用哪个工具，prompt 里已经告诉它了。

### 4. 明确列出反面模式

Claude Code 会显式地说*不要做什么*：

- 不要在行动前过度解释
- 不要列一堆选项让用户选
- 不要重复用户说过的话
- 不要给时间估计

这些"禁令"比正面指导更有效，因为它们精确地抑制了 LLM 的默认习惯。

## Codex CLI 的类似思路

OpenAI 的 Codex CLI 走了类似的路：

- 提供工作目录和环境信息
- 有 full-auto 模式——执行命令不需要确认
- System prompt 强调"自主执行，直接给结果"
- 沙盒机制保证安全，所以可以放心让 Agent 自主行动

Codex 的设计哲学是：*安全性通过沙盒保证，不通过反问保证*。Agent 可以放开手干，反正跑在沙盒里，出不了大事。

## 三层解法

总结下来，让 Agent 从"反问机器"变成"行动派"，需要三层设计：

*第一层：行为指令*

在 system prompt 里写清楚行为期望，而不是只描述能力：

```
错误：你有 read_file 工具可以读取文件。
正确：当用户问到文件内容时，直接使用 read_file 读取后回答，不要先问用户。
```

*第二层：上下文注入*

启动时注入工作环境信息（目录、配置路径、平台等），减少 Agent 的探索成本：

```
错误：让 Agent 自己发现 "数据在 ~/.xclaw/data/"
正确：system prompt 里直接写 "你的数据目录是 ~/.xclaw/data/"
```

*第三层：安全兜底*

Agent 敢于行动的前提是行动的后果可控。沙盒、权限白名单、确认机制应该在系统层面解决，而不是靠 Agent 自己"谨慎"来解决。

## 一个反直觉的结论

让 Agent 更聪明的方法，不是换更强的模型，而是写更好的 prompt。

Claude Code 用的是 Sonnet——不是最强的模型——但因为 system prompt 的设计足够精确，它比很多用 GPT-4/Opus 但 prompt 粗糙的 Agent 好用得多。

System prompt 不是"介绍信"，是"操作手册"。模型的能力是天花板，prompt 决定了它能触到天花板的多少。

---

> 在 AI Agent 的世界里，"聪明"不是模型参数决定的，是系统设计决定的。一个好的 system prompt 就像一本好的员工手册——它不让员工变得更聪明，但让聪明的员工知道该怎么干活。
