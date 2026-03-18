---
title: "拆解 gstack：Garry Tan 如何把 Claude Code 变成一支虚拟工程团队"
date: 2026-03-18
categories: [AI]
tags: [gstack, claude-code, ai-agent, workflow, playwright, bun]
---

> 如果说 Copilot 解决的是“帮我补代码”，那么 [gstack](https://github.com/garrytan/gstack) 试图解决的是另一个层级的问题：**怎么把 AI 编程从一个助手，升级成一套完整的软件生产系统。**

先说一个容易被忽略、但其实很重要的前提：**这个仓库的作者不是一个随便发 prompt 的 AI 博主，而是 Garry Tan。**

如果你对美国创业圈比较熟，他现在是 **Y Combinator 的 President & CEO**。在这之前，他做过 Posterous 联合创始人，早年在 Palantir 做过很早期的设计、产品和工程管理工作，还在 2013 年亲手做过 YC 内部的 Bookface。换句话说，这不是“一个会写 prompt 的人”在谈开发流程，而是一个长期横跨 **设计、产品、工程、管理和创业投资** 的人，把自己最新一轮 AI-native 工作方法直接开源出来。

所以读 `gstack` 时，我觉得不能只把它当成“又一个 agent 工具仓库”。更准确的阅读姿势是：**这是一个一线 builder 和顶级创业组织管理者，在 2026 年这个时间点公开展示自己如何用 Claude Code 组织软件生产。**

这两天我把 `gstack` 这个仓库系统读了一遍：README、[ARCHITECTURE.md](https://github.com/garrytan/gstack/blob/main/ARCHITECTURE.md)、[BROWSER.md](https://github.com/garrytan/gstack/blob/main/BROWSER.md)、[docs/skills.md](https://github.com/garrytan/gstack/blob/main/docs/skills.md)、若干 `SKILL.md`、`setup` 脚本，以及 `browse/src/*` 和 `scripts/gen-skill-docs.ts` 这几个关键源码文件。

读完后的结论很明确：

**gstack 不是一个 Agent 框架，也不是一组 prompt。它更像一个面向 Claude Code 的“AI 软件工厂操作系统”。**

它真正有意思的地方，不在于某一个命令，而在于它把下面这些东西拼成了一个统一系统：

- 持久化浏览器运行时
- 角色化的技能系统
- 标准化的 review gate
- 自动化的 QA 与 ship 流程
- 用模板生成的技能文档
- 一套非常强势、非常统一的工作哲学

## gstack 是什么

从仓库结构上看，`gstack` 可以粗略分成两层：

```
gstack
├── browse/               # 持久化浏览器：真正的运行时底座
├── plan-ceo-review/      # CEO 视角规划
├── plan-eng-review/      # 工程设计评审
├── plan-design-review/   # 设计评审
├── review/               # 代码审查
├── qa/                   # 真实浏览器 QA + 修 bug
├── ship/                 # 发版和 PR 流程
├── ...                   # 其他技能
└── scripts/gen-skill-docs.ts
```

它最核心的一句自我定义，写在架构文档里：

> The browser is the hard part — everything else is Markdown.

这句话基本点出了 `gstack` 的设计重心。

它不是在发明一个新的 agent runtime，而是在 Claude Code 已经有工具调用能力的前提下，补上两个真正缺的东西：

1. **一个足够快、足够稳定、足够持久的浏览器执行面**
2. **一套把 AI 开发流程组织起来的工作流外壳**

前者解决“Agent 能不能真的看见和操作应用”；后者解决“Agent 不是会不会写，而是该不该这样写、写完之后谁来 review、怎么 QA、什么时候 ship”。

## 第一层：浏览器守护进程才是它的技术底座

`gstack` 最扎实的工程部分，其实是 `/browse`。

很多 Agent 系统都支持浏览器，但常见问题也很一致：

- 每次调用都冷启动浏览器，慢
- 状态不持久，cookie 和登录态会丢
- 协议开销大，context token 浪费严重
- 工具连接脆弱，跑一会儿就断

`gstack` 的做法很直接：**不用 MCP，不走复杂协议，而是做一个本地常驻的 Chromium daemon。**

它的结构大概是这样：

```text
Claude Code
    │
    │ 调用 browse 二进制
    ▼
CLI compiled by Bun
    │ 读取 .gstack/browse.json
    │ POST /command
    ▼
Bun HTTP Server
    │
    │ 调 Playwright
    ▼
Headless Chromium
```

根据仓库里的实现：

- 首次启动大约 3 秒
- 后续命令大约 100-200ms
- 状态写在项目内的 `.gstack/browse.json`
- 浏览器 30 分钟空闲自动退出
- 端口随机选择，避免多工作区冲突
- 每个 session 用 bearer token 鉴权，只绑定 `localhost`

这套设计的重点不是“炫技”，而是很务实地围绕 Agent 使用模式优化。

Agent 做 QA 时不是只点一次页面，而是会连续执行二三十个操作：打开页面、读取元素、点击、输入、再截图、再查 console。如果每次都冷启动浏览器，体验会差一个数量级。`gstack` 的 daemon 模式本质上是在说：

**浏览器对 agent 来说不是一次性工具，而是长生命周期工作内存。**

### `@e1`、`@e2` 这种 ref 系统也很聪明

`gstack` 没让 agent 直接写 CSS selector 或 XPath，而是通过 `snapshot` 先读取 accessibility tree，再给元素分配 `@e1`、`@e2` 这种引用。

流程大概是：

1. 用 Playwright 读 ARIA snapshot
2. 给元素按树顺序分配 ref
3. 为每个 ref 建立 Playwright Locator
4. 后续 `click @e3` 时，直接解析到 Locator

这里有两个细节很见功力：

- 它**不修改 DOM**，避免 CSP、hydration、Shadow DOM 这些问题
- 它会检查 ref 是否 stale，元素不存在就立刻报错，让 agent 重新 `snapshot`

这比“往 DOM 打 label”那类实现要稳很多，也更像真正针对 agent 行为模式设计的工具。

## 第二层：技能系统不是 prompt，而是组织结构

如果只看 README，`gstack` 很容易被误解成“13 个 slash commands”。

但系统读下来会发现，重点根本不是 slash command 数量，而是它把 AI 开发过程强行拆成了几个**角色化认知模式**：

| 技能 | 扮演角色 | 目标 |
|---|---|---|
| `/plan-ceo-review` | CEO / Founder | 重定义问题，寻找更大的产品机会 |
| `/plan-eng-review` | Eng Manager | 把产品想法变成可构建的工程方案 |
| `/plan-design-review` | Senior Designer | 在实现前补齐界面、状态和体验细节 |
| `/review` | Staff Engineer | 找出 diff 里测试未必能发现的结构性问题 |
| `/qa` | QA Lead + Fix Engineer | 用真实浏览器测试，然后修 bug、补回归测试 |
| `/ship` | Release Engineer | 跑测试、检查 readiness、生成 PR |

这背后有个很重要的设计判断：

**AI 不是缺能力，而是缺“工作时该处于哪种思考模式”的外壳。**

普通 Agent 往往是一个人格从头聊到尾，于是产品讨论、架构设计、代码实现、质量控制全混在一起。`gstack` 则把这些阶段拆成不同技能，让 Claude 在不同阶段切换成不同角色。

这有点像把“工程团队分工”直接编码进系统里。

## 第三层：真正的创新点不是写代码，而是 review gate

我觉得 `gstack` 最值得学的，不是 `/browse`，而是它把 **review gate** 做成了默认工作流。

传统 vibe coding 最大的问题不是“代码写不出来”，而是：

- scope 会一路漂移
- 设计问题没人管
- edge case 没人补
- bug 往往要到上线后才暴露
- 文档和实现很快失真

`gstack` 的思路是：不要让“实现”直接成为第一步，而是先经过几道门。

### 1. CEO review：先问“你真的在解决对的问题吗”

`/plan-ceo-review` 最有意思的一点，是它不是帮你把需求做细，而是先挑战需求本身。

比如用户说“我要加图片上传”，它会追问：

- 真正的目标是不是“帮助卖家更快发出更好的商品信息”？
- 那是不是应该自动识别商品、补规格、生成 listing？
- “图片上传”本身也许只是一个 3 分功能，真正的 10 分功能在后面

这其实不是代码能力，而是**产品扩张能力**。

### 2. Engineering review：强迫系统显形

`/plan-eng-review` 反复强调 diagrams、failure modes、test matrix、observability。

这个方向我很认同。很多 LLM 在纯文字规划时会显得“很会说”，但一旦要求它画出状态机、数据流、错误路径，就会暴露出大量模糊地带。

`gstack` 的做法是把这些强制成输出要求，让 plan 不再只是“看起来合理”，而是要能落到系统边界和异常路径。

### 3. Review / QA / Ship：让质量控制进入默认路径

`/review` 不是泛泛 code review，而是 diff-scoped 地找：

- SQL / 数据安全
- race condition
- LLM trust boundary
- enum completeness
- frontend 设计反模式

`/qa` 更进一步，不只是报 bug，而是要求 agent：

1. 真实点击页面
2. 找到问题
3. 修复源码
4. 原子提交
5. 再次验证
6. 补回归测试

`/ship` 则把 readiness dashboard、review status、测试覆盖、版本号和 changelog 一起纳入发版流程。

这整套设计的潜台词是：

**AI 最容易放大的不是生产力，而是混乱。要想快，必须先把流程结构化。**

## 第四层：`SKILL.md` 不是说明文档，而是可执行工作规约

`gstack` 里另一个很巧的部分，是它对 `SKILL.md` 的处理。

很多人把技能文件当 prompt 文本，但 `gstack` 明显不是这样看。它把 skill 当成一种**介于代码、流程、文档之间的执行规约**。

从 `scripts/gen-skill-docs.ts` 可以看出，它不是让每个技能作者手写文档，而是走一套模板系统：

```text
SKILL.md.tmpl
   ↓
读取 commands.ts / snapshot.ts / 公共前置逻辑
   ↓
生成最终 SKILL.md
```

这样做有几个好处：

- 命令文档和真实实现共用同一份元数据，减少漂移
- 每个 skill 自动共享相同的 preamble、提问规范、setup 检查
- CI 可以验证生成结果是否过期

这其实是在把 prompt engineering 往 **docs-as-code** 的方向推进。

不是“我写了一段 prompt，希望别过期”，而是“我的技能说明是从源码生成的，因此可以被测试、被校验、被 diff”。

## 第五层：它最强势的地方，是统一工作哲学

如果只看技术，`gstack` 当然做得不错；但它真正最强的，是它几乎把创作者的工作方法整包编码进去了。

最明显的是两个公共机制。

### 1. 统一 preamble

几乎所有技能前面都会先做这些事：

- 检查版本升级
- 记录会话
- 获取当前 branch
- 输出当前上下文
- 进入统一的 AskUserQuestion 格式

这意味着每个 skill 不是一个孤立 prompt，而是共享一个统一的“组织语境”。

### 2. Boil the Lake / Completeness Principle

这是 `gstack` 最近版本反复强调的核心哲学：**AI 把实现成本压低后，默认应该选择“更完整”的方案，而不是“差不多够用”的方案。**

它会在 skill 里明确要求：

- 评估方案时偏向完整实现
- 不要因为只多几十行代码就砍掉 edge case
- 测试不要拖到后面再补
- 估算工期时同时给出 human 时间和 CC+gstack 时间

这套哲学不一定永远对，但它非常清楚地解释了 `gstack` 的气质：它不是一个“保守节制”的系统，而是一个**假设 AI 让完整性变便宜，因此应该主动追求 completeness** 的系统。

## 为什么说 gstack 更像“AI 软件工厂”

读完整个仓库后，我最强的感受是：`gstack` 在试图定义的，不是“更强的 coding assistant”，而是 **AI-native 的工程组织方式**。

传统开发栈大概是：

```text
编辑器 + Git + CI + 测试 + 浏览器 + code review + PM/设计/QA 协作
```

而 `gstack` 想把其中一大块压缩进 Claude Code 内部：

```text
Claude Code
  + 角色化技能
  + 持久浏览器
  + review gate
  + QA 回路
  + ship 流程
```

也就是说，它想把“一个团队的工作流”收敛成“一个人驾驭多种 agent 角色的操作台”。

这也是为什么 README 一直在讲 “virtual engineering team”。

这不是比喻，几乎是它的产品定义本身。

## 我觉得它最值得借鉴的 4 个点

如果把 `gstack` 和具体生态绑定部分分开，我觉得最值得借鉴的是这四点：

### 1. 浏览器要常驻，不要每次冷启动

只要 Agent 真的要做 QA 或 dogfooding，浏览器就应该被当成长期状态，而不是一次性工具调用。

### 2. 技能最好是“角色切换器”，不是“功能菜单”

真正有效的 skill 不是“你可以做 X”，而是“你现在要用什么视角思考这件事”。

### 3. 文档必须从实现生成，不能靠手写同步

Agent 系统尤其容易文档漂移。把 prompt / skill 文档生成化、测试化，是非常值得抄的做法。

### 4. 速度必须靠 gate 来守住，不然只会更乱

AI 编码越快，越需要结构化 review。否则你得到的不是 10x 工程效率，而是 10x 混乱制造速度。

## 它的局限也很明显

`gstack` 当然也不是没有边界。

### 1. 它强绑定 Claude Code 生态

目录结构、`CLAUDE.md`、slash command、skill 发现机制，全都围绕 Claude Code。想原样迁移到别的 agent 里，并不现实。

### 2. 它高度依赖 prompt discipline

很多能力本质上来自 skill 约束，而不是独立 runtime 能力。这让它很灵活，但也意味着行为质量高度依赖 prompt 维护质量。

### 3. 它非常“Garry Tan 化”

它的语气、工作节奏、scope 倾向、完整性偏好，都带有非常强的作者方法论。你很难说这是通用最优解，但它确实是一套风格鲜明、几乎不可误认的系统。

不过换个角度看，这恰恰也是它有价值的地方：它不是中性的框架，而是一套明确表达 worldview 的工程操作法。

## 总结

如果只把 `gstack` 看成“一个带浏览器的 Claude Code skill 集合”，会低估它。

更准确的说法是：

**gstack 在尝试把“AI 编码”从单点能力，推进成一套包含产品、设计、工程、QA、发版的完整生产系统。**

它最重要的发明不是某个命令，而是一个观念：

> 当模型足够强、工具足够快之后，瓶颈不再是“AI 会不会写”，而是“你有没有把整个软件生产流程重新组织一遍”。

这也是为什么我觉得这个仓库值得认真读。  
它不只是展示“Agent 能做什么”，而是在展示：**一个人如何用 Agent 重新搭建自己的软件工厂。**
