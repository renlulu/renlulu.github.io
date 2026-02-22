---
title: "AGENTS.md 实践指南：如何给 AI Agent 写一份好的「入职文档」"
date: 2026-02-22
categories: [AI]
tags: [agents-md, coding-agent, codex, claude-code, cursor, copilot]
---

> AI coding agent 越来越强，但它不了解你的项目约定。AGENTS.md 就是解决这个问题的：一份给 agent 看的、机器可读的项目指南。

## 问题的起源

用过 AI coding agent 的人一定遇到过这类问题：

- 项目用 pnpm，agent 偏要 `npm install`
- 团队约定 commit message 用 Conventional Commits，agent 写了个 `fix stuff`
- 代码库有自定义的 lint 规则，agent 生成的代码全是 warning
- monorepo 里有 5 个子项目，agent 在错误的目录下运行测试

这些问题的根源是一样的：**agent 不知道你的项目约定**。它知道 TypeScript 的语法，知道 React 的 API，但它不知道你的团队用 pnpm、commit message 要加前缀、lint 配置关了分号。

人类新成员入职时，我们会给一份 onboarding 文档。AI agent 也需要一份。

## AGENTS.md 是什么

[AGENTS.md](https://agents.md/) 是一个简单的 Markdown 文件，放在项目根目录，给 AI coding agent 提供项目上下文和指令。可以理解为：**README 是给人看的，AGENTS.md 是给 agent 看的**。

它的设计哲学很直接：

- **普通 Markdown**，不需要任何特殊语法
- **Git 管理**，和代码一起版本控制
- **层级化**，根目录一份全局的，子目录可以有专属的
- **工具无关**，不绑定任何特定的 agent

目前已有 [60,000+ 开源项目](https://agents.md/) 采用，支持的 agent 包括 OpenAI Codex、Claude Code（通过 CLAUDE.md）、Cursor、GitHub Copilot、Google Jules 等 20+ 工具。

### 起源：从 Codex 的 commit 历史看

在[《解剖 3913 个 Commit：OpenAI Codex 从 0 到 1 的构建之路》](/posts/codex-commit-history-analysis/)中，我们分析了 Codex 的完整 Git 历史。其中有一个关键节点：

> **2025 年 5 月 9 日**（commit [4aacd80fa](https://github.com/openai/codex/commit/4aacd80fa)）：AGENTS.md 取代 instructions.md。

这意味着 Codex 团队在开源后第 23 天就意识到，agent 需要一种标准化的方式来接收项目指令。`instructions.md` 是最初的尝试，`AGENTS.md` 是标准化后的结果。

## 一个真实的例子：Codex 自己的 AGENTS.md

与其空谈规范，不如看看 [OpenAI Codex 仓库自己的 AGENTS.md](https://github.com/openai/codex/blob/main/AGENTS.md) 怎么写的。这个文件大约 200 行，覆盖了以下内容：

### 1. 命名约定和代码风格

```markdown
- Crate names are prefixed with `codex-`.
  For example, the `core` folder's crate is named `codex-core`
- When using format! and you can inline variables into {}, always do that.
- Always collapse if statements (clippy::collapsible_if)
- Always inline format! args when possible (clippy::uninlined_format_args)
```

直接告诉 agent：crate 名有前缀，format! 要内联变量，if 语句要折叠。这些是人类 code review 时一定会指出的东西，写在 AGENTS.md 里让 agent 一开始就遵守。

### 2. 禁区声明

```markdown
- Never add or modify any code related to
  `CODEX_SANDBOX_NETWORK_DISABLED_ENV_VAR` or `CODEX_SANDBOX_ENV_VAR`.
```

这种"绝对不要碰"的指令很重要。agent 不知道某些代码有特殊用途（如沙箱环境变量），明确禁止可以避免 agent 好心办坏事。

### 3. 构建和测试流程

```markdown
Run `just fmt` automatically after you have finished making Rust code changes;
do not ask for approval to run it.

1. Run the test for the specific project that was changed.
   For example, if changes were made in `codex-rs/tui`,
   run `cargo test -p codex-tui`.
2. Once those pass, if any changes were made in common, core, or protocol,
   run the complete test suite with `cargo test`.
```

这段指令精确到了流程步骤：先格式化（不需要确认），然后跑局部测试，最后按需跑全量测试。

### 4. 框架特定约定

```markdown
### TUI Styling (ratatui)
- Prefer Stylize helpers: use "text".dim(), .bold(), .cyan()
  instead of manual Style where possible.
- Avoid hardcoded white: do not use `.white()`;
  prefer the default foreground (no color).
```

对 ratatui（Rust TUI 框架）的具体用法给出了明确偏好，这种信息不可能出现在框架官方文档里，只有项目内部才知道。

### 5. 子目录 AGENTS.md

Codex 仓库在 `codex-rs/tui/src/bottom_pane/` 下还有一个专属的 AGENTS.md：

```markdown
# TUI bottom pane (state machines)
When changing the paste-burst or chat-composer state machines, keep docs in sync:
- Update the relevant module docs
- Update the narrative doc `docs/tui-chat-composer.md`
- Keep implementations/docstrings aligned
```

这体现了 AGENTS.md 的**层级化设计**：根目录的管全局，子目录的管局部。agent 编辑这个子目录的文件时，两份都会被加载。

## 级联规则：层级如何生效

AGENTS.md 的级联规则用一句话概括：**越近的越优先，越明确的越优先**。

以 Codex（OpenAI 的 agent）为例，它加载指令的[完整流程](https://developers.openai.com/codex/guides/agents-md/)是：

| 优先级 | 来源 | 说明 |
|--------|------|------|
| 1（最低） | `~/.codex/AGENTS.md` | 全局默认（个人偏好） |
| 2 | 仓库根目录 `AGENTS.md` | 项目约定 |
| 3 | 子目录 `AGENTS.md` | 子项目/模块专属规则 |
| 4（最高） | 用户直接输入的 prompt | 当前任务的即时指令 |

在同一级目录下，`AGENTS.override.md` 优先于 `AGENTS.md`。这让你可以临时覆盖团队约定而不删除原文件。

**关键限制：**

- 每个目录最多加载一份文件
- 所有文件合计不超过 32 KiB（可通过 `project_doc_max_bytes` 配置调大）
- 空文件会被跳过

## 写一份好的 AGENTS.md：实践指南

### 原则一：精简为王

[Vercel 的评测数据](https://vercel.com/blog/agents-md-outperforms-skills-in-our-agent-evals)给出了一个关键发现：

> 一份压缩到 **8KB 的 AGENTS.md**，在 Next.js 16 API 的评测中达到了 **100% 通过率**，而没有任何文档引导时只有 53%。

| 配置 | 通过率 |
|------|--------|
| 无文档 | 53% |
| Skills（默认） | 53% |
| Skills + 明确指令 | 79% |
| AGENTS.md 文档索引 | **100%** |

8KB 就够了。原始文档有 40KB，压缩 80% 后效果反而更好。**核心教训：agent 不需要百科全书，它需要精准的操作手册**。

### 原则二：最小必要信息

根据 [AI Hero 的完整指南](https://www.aihero.dev/a-complete-guide-to-agents-md)，根目录 AGENTS.md 只需要三样东西：

```markdown
# MyProject

一句话描述：MyProject 是一个 Next.js 电商平台，使用 Prisma + PostgreSQL。

## 包管理

使用 pnpm，不要使用 npm 或 yarn。

## 非标准命令

- 类型检查：`pnpm typecheck`（不是 tsc）
- Lint：`pnpm lint:fix`
- 测试：`pnpm test:unit`（单元测试）、`pnpm test:e2e`（端到端）
```

就这些。不需要写架构说明，不需要列目录结构，不需要解释业务逻辑。agent 会自己探索代码库——你只需要告诉它那些**它自己发现不了的东西**。

### 原则三：指令要可执行

**坏的写法：**

```markdown
请确保代码质量良好，遵循最佳实践。
```

**好的写法：**

```markdown
- 每个 PR 前运行 `pnpm lint:fix && pnpm typecheck`
- 新增 API endpoint 必须在 `tests/api/` 下有对应的测试文件
- 错误处理使用 `AppError` 类型，不要用裸的 `throw new Error()`
```

agent 需要的是**可执行的指令**，不是抽象的原则。

### 原则四：用禁区代替偏好

与其告诉 agent 你喜欢什么，不如告诉它**绝对不要做什么**：

```markdown
## 禁止事项

- 不要修改 `src/core/auth/` 下的任何文件，这是安全关键路径
- 不要升级 React 大版本（当前锁定 18.x）
- 不要在 `utils/` 下创建新文件，除非现有工具函数确实不够用
- 不要使用 `any` 类型
```

agent 有"改进"代码的冲动。禁区声明可以防止它好心办坏事。

### 原则五：渐进式信息加载

对于复杂项目，不要把所有规则塞进一个文件。用层级结构实现**渐进式加载**：

```
AGENTS.md                    → 全局：包管理、通用命令
packages/api/AGENTS.md       → API 子项目：路由约定、中间件规则
packages/web/AGENTS.md       → Web 子项目：组件命名、样式约定
packages/shared/AGENTS.md    → 共享库：发布流程、API 稳定性要求
```

每个文件聚焦自己的职责。agent 编辑 `packages/api/` 下的文件时，会同时加载根目录和 `packages/api/` 的 AGENTS.md——不需要把 API 规则写到全局文件里。

### 模板：一个通用的起点

```markdown
# [项目名]

[一句话描述项目是什么、用了什么技术栈]

## 开发命令

- 安装依赖：`pnpm install`
- 开发服务器：`pnpm dev`
- 构建：`pnpm build`
- 测试：`pnpm test`
- Lint：`pnpm lint:fix`

## 代码约定

- [列出 2-3 条最重要的、agent 不容易自己发现的约定]

## 禁止事项

- [列出绝对不要做的事]

## 测试要求

- [新功能/修复需要什么级别的测试覆盖]
```

**不要超过 50 行**。如果超了，考虑拆分到子目录。

## 常见错误

### 1. 自动生成

让 agent 自己生成 AGENTS.md 是常见做法，但效果不好。自动生成的文件倾向于面面俱到，把显而易见的东西也写进去（"使用 TypeScript 编写代码"），真正重要的项目特殊约定反而被淹没。

### 2. 文档化目录结构

```markdown
## 目录结构
src/
├── components/
├── pages/
├── utils/
...
```

这种信息变化频繁、很快就会过期，而且 agent 可以通过 `ls` 自己发现。不要写。

### 3. 反应式累积

每次 agent 犯错就加一条规则，最终导致文件膨胀、规则互相矛盾。应该定期审查，合并重复规则，删除过时条目。

### 4. 绝对化语言过多

```markdown
- ALWAYS use exactly 2 spaces for indentation
- NEVER use default exports
- MUST add JSDoc to every function
```

全是 ALWAYS/NEVER/MUST 时，agent 反而无法判断优先级。保留给真正关键的规则（如安全约束），其他用正常语气即可。

## 各家 Agent「入职文档」对比

AGENTS.md 不是唯一的方案。以下是各主要 agent 的原生配置文件对比：

| 特性 | AGENTS.md | CLAUDE.md | .cursor/rules/ | copilot-instructions.md |
|------|-----------|-----------|-----------------|------------------------|
| 格式 | Markdown | Markdown | .mdc（带 glob） | Markdown |
| 层级化 | 目录级联 | 目录级联 | 按 glob 匹配 | 仓库级 + .instructions.md |
| override 文件 | 支持 | 不支持 | N/A | N/A |
| 大小限制 | 32 KiB（可配置） | 无硬限制 | 无硬限制 | 无公开限制 |
| 工具绑定 | 工具无关 | Claude Code | Cursor | GitHub Copilot |
| Git 管理 | 是 | 是 | 是 | 是 |
| 动态加载 | 否（每次会话加载） | 否 | 支持按文件类型自动/手动触发 | 自动附加 |

**实际建议：两份都写，或者用软链接**

如果你的团队同时使用多个 agent（这很常见），推荐方案：

```bash
# 方案一：软链接
ln -s AGENTS.md CLAUDE.md

# 方案二：CLAUDE.md 引用 AGENTS.md
echo "请同时参考 AGENTS.md 中的项目指南。" > CLAUDE.md
```

## 安全风险：不可忽视的攻击面

AGENTS.md 的便利性背后有一个严肃的安全问题：**它是 prompt injection 的天然载体**。

### Rules File Backdoor 攻击

2025 年 3 月，安全公司 [Pillar Security 披露了一种攻击手法](https://www.pillar.security/blog/new-vulnerability-in-github-copilot-and-cursor-how-hackers-can-weaponize-code-agents)：在 `.cursorrules`、`copilot-instructions.md` 等配置文件中嵌入 Unicode 零宽字符，隐藏恶意指令。

**攻击原理：**

1. 攻击者在配置文件中嵌入 Unicode 零宽字符（如 U+200B 零宽空格、U+E0000-U+E007F Unicode Tag 字符）
2. 这些字符人类在 code review 时看不到，但 LLM 可以解析执行
3. 隐藏的指令可以让 agent 在生成的代码中注入恶意内容（如 `<script>` 标签）
4. agent 不会在聊天记录中提及这些隐藏操作

**影响范围：**

所有通过文件加载 system prompt 的 agent 都受影响，包括 AGENTS.md、CLAUDE.md、.cursorrules、copilot-instructions.md。

**防御建议：**

- **审查 PR 时检查配置文件的原始字节**：`xxd AGENTS.md | head` 或使用编辑器的"显示不可见字符"功能
- **配置 CI 检查**：检测配置文件中的零宽字符和非 ASCII 字符
- **限制 AGENTS.md 的修改权限**：使用 CODEOWNERS 限制谁能修改
- **不要信任外部仓库的 AGENTS.md**：clone 未知仓库后，先检查配置文件

```bash
# 检测文件中的零宽字符和不可见 Unicode
perl -ne 'print "$.: $_" if /[\x{200B}-\x{200F}\x{2028}-\x{202F}\x{2060}-\x{206F}\x{E0000}-\x{E007F}\x{FEFF}]/' AGENTS.md
```

## AAIF：标准化的未来？

2025 年底，Linux Foundation 成立了 [Agentic AI Foundation（AAIF）](https://www.linuxfoundation.org/press/linux-foundation-announces-the-formation-of-the-agentic-ai-foundation)，将三个核心项目收入旗下：

1. **MCP（Model Context Protocol）**：agent 与工具的通信协议
2. **AGENTS.md**：agent 的项目指引规范
3. **Goose**：开源 agent 框架

共同创始成员包括 AWS、Anthropic、Google、Microsoft、OpenAI 等科技巨头。

这意味着 AGENTS.md 正在从一个社区约定走向行业标准。但它能否成为真正的统一标准，取决于：

- Claude Code 是否会支持直接读取 AGENTS.md（目前需要通过 CLAUDE.md 引用）
- Cursor 是否会放弃自己的 .mdc 格式转向 AGENTS.md
- 更多 IDE 和 agent 是否会内置支持

在那之前，写两份文件（AGENTS.md + 你用的 agent 的原生格式）仍然是最稳妥的做法。

## 总结

AGENTS.md 的核心价值是一句话：**把你会在 code review 中反复指出的问题，提前告诉 agent**。

不需要写很多——8KB 以内，50 行左右，聚焦于 agent 自己发现不了的项目约定。用可执行的指令代替抽象原则，用禁区声明代替偏好建议，用目录层级代替巨型文件。

最后，别忘了把它当代码一样维护：定期审查、删除过时规则、保持精简。一份过期的 AGENTS.md 比没有更糟糕——因为 agent 会认真执行每一条指令，包括错误的。
