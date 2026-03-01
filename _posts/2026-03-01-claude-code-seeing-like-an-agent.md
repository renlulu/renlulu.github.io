---
title: "像 Agent 一样思考：Claude Code 团队的工具设计心法"
date: 2026-03-01 07:00:00 +0800
categories: [AI]
tags: [Claude Code, Agent, Tool Design, Anthropic]
---

Anthropic 的 Thariq Shihipar 最近发了一篇 [Lessons from Building Claude Code: Seeing like an Agent](https://x.com/trq212/status/2027463795355095314)，分享了 Claude Code 团队在工具设计上的踩坑和心得。这是他 "Lessons from Building Claude Code" 系列的第二篇（第一篇是 [Prompt Caching Is Everything](https://x.com/trq212/status/2024574133011673516)）。

核心观点只有一句：**构建 Agent 最难的部分不是模型能力，而是设计它的行动空间（action space）。**

## 什么是行动空间？

Claude 通过 Tool Calling 来执行操作。API 层面提供了多种工具构建方式——bash、skills、code execution 等原语。但"有哪些工具可以调"只是一半的问题，另一半是"这些工具应该长什么样"。

Thariq 用了一个很直觉的比喻：

> 想象你在解一道很难的数学题。给你一张纸，你能算；给你计算器，好一些；给你一台电脑能跑代码——那就是质变。

关键在于：**工具要和模型自身的能力匹配**。给 Agent 的工具不够强，它发挥不出来；给的工具太复杂或语义混乱，它反而搞不定。

但怎么知道什么工具匹配？答案是——**像 Agent 一样看世界**。不是你觉得什么工具好，而是**观察模型的输出、阅读它的行为、不断实验**，看它在什么样的工具设计下表现最好。

> "Understanding what tools to give requires you to pay attention, read outputs, and experiment."

## 案例一：AskUserQuestion 的三次迭代

**问题：** Claude Code 需要更好地向用户提问。写代码的时候经常需要澄清需求，但最初 Claude 要么不问，要么问的方式很低效——用户得打一大段文字来回答纯文本问题，摩擦很大。

**目标：** 降低沟通摩擦，提高信息获取的带宽。

### 第一次尝试：复用 ExitPlanTool

在 ExitPlanTool（退出规划模式的工具）里加一个 `questions` 参数，让 Claude 在退出规划时顺便提问。

结果：**Claude 搞混了。** ExitPlanTool 的语义是"我规划完了"，现在又要它同时表达"我有问题要问"，两件事混在一起，Claude 的行为变得不稳定——有时候该问不问，有时候不该退出规划模式却退了。

**教训：一个工具塞两个语义，模型会困惑。**

### 第二次尝试：用 Markdown 格式输出

不加新工具，而是在系统提示里告诉 Claude：用特定的 markdown 格式输出问题列表。

Claude "似乎可以很好地输出这个格式"——但**不能保证**。有时候它会多加几句话，有时候格式完全跑偏。没有工具约束的结构化输出，本质上就是在赌模型的遵从性。

**教训：自由文本输出不可靠，结构化需要工具级别的约束。**

### 第三次（最终方案）：独立的 AskUserQuestion 工具

做一个全新的、独立的 `AskUserQuestion` 工具：

- Claude 可以在**任何时刻**调用，不限于规划模式
- 工具强制结构化输出：必须给用户提供**多个选项**而不是开放式问题
- 用户点选选项而不是打字，大幅降低回答摩擦

效果：Claude **天然倾向于使用**这个工具，不需要反复在系统提示里催它。Thariq 的原话：

> "Claude seems to like calling this tool."

这句话很有意思——当工具设计和模型的"直觉"对齐时，你不需要费力引导，模型自己就会用。

**核心启示：工具的语义要清晰、单一。** 一个工具干一件事，比一个工具干三件事靠谱得多。结构化的工具调用比自由格式的文本输出可控得多。

## 案例二：Todo → Task 的进化

### 早期：TodoWrite + 系统提醒

早期 Claude Code 有一个 `TodoWrite` 工具，帮模型追踪要做的事情——简单的 checklist。团队还每隔 5 步插入一条系统提醒："别忘了更新你的 todo list。"

这在早期模型上是**必要的**。没有这些提醒，模型干着干着就忘了原来的目标，跑偏了。

### 转折：模型变强了

但随着模型能力提升（尤其是 Opus 4.5），问题反转了：

> **"As models improve, tools that were once necessary might now be holding them back."**

新模型不仅不需要每 5 步提醒，提醒反而**干扰了它的正常工作流**。模型自己能记住目标，你插一条"别忘了更新 todo"进来，它反而要花精力处理这个打断。

另外，Opus 4.5 在使用子代理（sub-agents）方面表现明显更好——它能有效地把任务分发给子代理，协调多个 Agent 并行工作。这意味着 Todo 的简单 checklist 模式已经不够用了。

### 升级：Task 系统

团队用 `Task` 工具替代了 `TodoWrite`：

> "Todos were meant to keep the model on track. Tasks help agents communicate with each other."

| 特性 | TodoWrite（旧） | Task（新） |
|------|-----------------|------------|
| 定位 | 帮模型记住要做什么 | 帮多个 Agent 协调工作 |
| 结构 | 扁平列表 | 支持依赖关系（blocks/blockedBy） |
| 协作 | 单 Agent 使用 | 跨子代理共享更新 |
| 生命周期 | 只能添加和勾选 | 模型可以修改、删除、重新排序 |
| 提醒 | 每 5 步系统提醒 | 模型自主管理，无需提醒 |

**核心教训：工具设计不是一劳永逸的，要随模型能力同步演进。** 你给弱模型的拐杖，在强模型手里可能变成枷锁。团队需要定期审视：这个工具还合适吗？模型是不是已经超越了它？

## 案例三：从 RAG 到自主探索

### 第一阶段：RAG 向量数据库

代码搜索是 Agent 的核心能力之一。Claude Code 最初用的是经典方案——**RAG 向量数据库**：把代码库向量化，用户提问时检索相关片段，喂给模型作为上下文。

问题：
- 需要预先索引，设置成本高
- 在不同开发环境中**可能很脆弱**（Thariq 原话："could be fragile across different environments"）
- 检索结果是"系统觉得相关的"，不一定是"Agent 此刻需要的"

### 第二阶段：给模型搜索工具

后来团队换了思路：**不再帮 Agent 找上下文，而是给它工具让它自己找。**

给 Claude Grep、Glob、Read 等搜索工具，让它自行搜索代码库、构建上下文。这个转变背后有一个越来越明确的规律：

> "Claude越聪明，在获得正确工具时就越能很好地构建自己的上下文。"

### 第三阶段：渐进式信息披露

Agent Skills 的引入正式确立了**渐进式信息披露（Progressive Disclosure）** 的概念：

1. Agent 先用 Glob 找到可能相关的文件
2. 用 Read 读取文件内容
3. 文件内容中引用了其他文件（比如 `CLAUDE.md` 里引用了 `docs/architecture.md`）
4. Agent 递归跟踪引用，逐层深入
5. 最终找到真正需要的信息

这就像人类程序员的工作方式——你不会一上来就把整个代码库读完，而是顺着线索一层一层往下找。

Thariq 对这个进步的描述：

> "在一年的过程中，Claude 从基本无法构建自己的上下文，发展到能够跨多层文件进行嵌套搜索以找到所需的确切内容。"

**核心洞察：** 与其帮 Agent 做上下文检索（RAG），不如给它检索的能力让它自己来。模型越强，这个策略越有效。

## 案例四：子代理实现功能扩展

### 困境：工具膨胀

Claude Code 大约有 **20 个工具**。团队对此非常克制：

> "添加新工具的门槛很高。"

每加一个工具：
- 系统提示变长 → 上下文更嘈杂
- prompt cache 命中率可能下降 → 延迟和成本上升
- 模型需要理解的选项增多 → 决策质量可能下降

### 问题：用户不知道怎么用 Claude Code

Claude 对自己的工具"了解不够"——用户问"怎么设置 hooks"或"怎么用 MCP"，Claude 经常答不上来。

方案 A：加一个文档搜索工具 → 工具数 +1，污染主上下文

方案 B：把所有文档塞进系统提示 → 大量 token 消耗，而且绝大部分时候用不到。更严重的是会**增加"上下文腐烂"（context rot）**——无关信息太多，影响模型在主要任务（写代码）上的表现。

方案 C（最初尝试）：给 Claude 一个文档链接，让它自己去查 → 问题是 Claude 会把**大量搜索结果**都加载到上下文里，即使只需要一个简短答案。

### 最终方案：Claude Code Guide 子代理

创建一个专门的 **Claude Code Guide 子代理**。当用户问"Claude Code 怎么用"类的问题时：

1. 主 Agent 识别这是使用问题
2. 把任务委托给 Guide 子代理
3. 子代理有独立的系统提示，包含**关于如何搜索文档的详细指令**
4. 子代理搜索、找到答案、返回给主 Agent
5. 主 Agent 的上下文保持干净

**功能扩展了，但主 Agent 的工具数量没有增加。** 子代理的搜索过程和中间结果不会污染主 Agent 的上下文。

这是一个值得学习的架构模式：**能力需要扩展时，优先考虑子代理，而不是加工具。**

## 总结：没有银弹，只有方法论

Thariq 在结尾很诚实地说：

> "如果你期望一套关于如何构建工具的严格规则，遗憾的是这个指南不会提供。"

工具设计**既是艺术也是科学**，高度依赖于三个变量：
- **使用的模型** — 不同模型的能力和倾向不同
- **Agent 的目标** — 写代码 vs 搜索信息 vs 与用户沟通，需要不同的工具
- **运行环境** — 本地开发 vs 云端 vs 受限环境

从四个案例可以提炼出 Claude Code 团队的工具设计原则：

1. **语义单一** — 一个工具做一件事，不要混合职责。模型对语义冲突很敏感（AskUserQuestion 案例）
2. **随模型进化** — 定期审视工具是否还合适，拐杖不要变枷锁。模型能力在快速提升，去年的最佳实践今年可能是反模式（Todo → Task 案例）
3. **渐进披露** — 给 Agent 探索的能力，而不是一次性灌输信息。模型越强，自主探索越优于预检索（搜索案例）
4. **子代理优先于工具** — 能力扩展优先考虑子代理，控制工具膨胀，保持主上下文干净（Guide 案例）

底层方法论就一句话：

> **"经常实验、阅读输出、尝试新事物。像 Agent 一样看待问题。"**

不是你觉得什么设计好，而是看模型在你的设计下表现怎么样。好的工具设计是观察出来的，不是想出来的。

---

*本文基于 Thariq Shihipar ([@trq212](https://x.com/trq212)) 的推文线程 [Lessons from Building Claude Code: Seeing like an Agent](https://x.com/trq212/status/2027463795355095314) 整理，全文可在 [Techtwitter](https://www.techtwitter.com/articles/lessons-from-building-claude-code-seeing-like-an-agent) 阅读。推荐同系列第一篇 [Prompt Caching Is Everything](https://x.com/trq212/status/2024574133011673516)。*
