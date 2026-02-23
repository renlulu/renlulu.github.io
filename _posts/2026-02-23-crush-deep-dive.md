---
title: "深入 Crush 源码：Charm 团队如何用 Go 打造一个 Coding Agent"
date: 2026-02-23 01:00:00 +0800
categories: [AI]
tags: [coding-agent, crush, go, charm, open-source]
mermaid: true
---

> 读完 Crush 的 263 个 Go 源文件后，我发现了几个出乎意料的设计决策。这不是一篇架构介绍，而是一次对工程细节的深挖。

## 背景

Crush 是 Charm 团队开源的终端 AI 编码助手，前身是 TermAI（2025 年 3 月）和 OpenCode。3211 个 commit，127 个版本，20,300+ stars。本文不再重复"它有哪些功能"——那些看 README 就够了。我想聊的是：**读完源码后，哪些设计让我觉得"原来如此"或者"居然这样做"**。

## 发现一：它没有 Shell，它是一个 Shell

Crush 的 bash 工具不是你想象的那样工作的。

大多数 coding agent 执行 bash 命令的方式很直觉：`exec.Command("bash", "-c", command)` 启动一个子进程。Claude Code 这样做，Codex 也这样做（只是包了一层沙箱）。

Crush 不是。它内嵌了一个 **Go 实现的 POSIX Shell 解释器**（[mvdan.cc/sh/v3](https://github.com/mvdan/sh)）：

```go
// internal/shell/shell.go
import (
    "mvdan.cc/sh/moreinterp/coreutils"
    "mvdan.cc/sh/v3/interp"
    "mvdan.cc/sh/v3/syntax"
)

func (s *Shell) execCommon(ctx context.Context, command string, stdout, stderr io.Writer) error {
    // 先解析成 AST
    line, _ := syntax.NewParser().Parse(strings.NewReader(command), "")
    // 再用解释器执行
    runner, _ := s.newInterp(stdout, stderr)
    return runner.Run(ctx, line)
}
```

这意味着什么？

**1. 命令在 AST 层被拦截，不是在字符串层**

安全命令过滤不是用正则匹配字符串，而是在解释器的 `ExecHandler` 链中拦截。`blockHandler` 在命令被执行之前检查参数：

```go
func (s *Shell) blockHandler() func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
    return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
        return func(ctx context.Context, args []string) error {
            for _, blockFunc := range s.blockFuncs {
                if blockFunc(args) {
                    return fmt.Errorf("command is not allowed: %q", args[0])
                }
            }
            return next(ctx, args)
        }
    }
}
```

用字符串匹配拦截 `curl` 很容易被绕过（`\curl`、`$(echo curl)`），但在 AST 层拦截，解释器已经解析完管道、替换、转义，拿到的是最终要执行的命令和参数。

**2. 环境变量和工作目录跨命令持久化**

每次执行后，Shell 从解释器同步状态：

```go
func (s *Shell) updateShellFromRunner(runner *interp.Runner) {
    s.cwd = runner.Dir
    s.env = s.env[:0]
    for name, vr := range runner.Vars {
        if vr.Exported {
            s.env = append(s.env, name+"="+vr.Str)
        }
    }
}
```

`cd /tmp && export FOO=bar` 之后，下一条命令自动在 `/tmp` 目录下执行，且能看到 `$FOO`。真正的 shell 体验。

**3. Windows 上也跑 POSIX Shell**

在 Windows 上，Crush 启用 Go 实现的 coreutils（`ls`、`cat`、`grep` 等的 Go 纯实现），让 POSIX 命令在没有 bash 的环境下也能工作：

```go
// internal/shell/coreutils.go
func init() {
    if v, err := strconv.ParseBool(os.Getenv("CRUSH_CORE_UTILS")); err == nil {
        useGoCoreUtils = v
    } else {
        useGoCoreUtils = runtime.GOOS == "windows"
    }
}
```

这就是为什么 Crush 能支持 8 个 OS（包括 Android 和 BSD）——它不依赖宿主系统的 shell。代价是和真实 bash 有行为差异，但换来了真正的跨平台一致性。

## 发现二：1 分钟自动升级为后台任务

Crush 的 bash 工具有一个精巧的执行模式：命令开始时是同步的，但如果超过 1 分钟还没完成，**自动升级为后台任务**。

```go
const AutoBackgroundThreshold = 1 * time.Minute

// 同步执行，等待完成或超时
for {
    select {
    case <-ticker.C:
        stdout, stderr, done, execErr = bgShell.GetOutput()
        if done { break waitLoop }
    case <-timeout:   // 1 分钟到了
        break waitLoop
    case <-ctx.Done(): // 用户取消
        bgManager.Kill(bgShell.ID)
        return
    }
}

if !done {
    // 还在跑——保持为后台任务，返回 job ID
    response := "Command has been moved to background.\nBackground shell ID: " + bgShell.ID
}
```

更妙的是开头的"快速失败检测"：对于显式后台任务，先等 1 秒检查是否立即失败（被禁命令、语法错误等），再告诉用户"后台任务已启动"。这避免了用户拿到一个已经挂了的 job ID。

同时还有上限控制：最多 50 个并发后台任务，完成的任务保留 8 小时后自动清理。

## 发现三：Provider 抽象的真实代价

Crush 通过 Fantasy 库支持 14+ 个 LLM provider。听起来很美，但源码揭示了一个关键问题：**provider 抽象必然泄漏**。

最典型的例子是 `workaroundProviderMediaLimitations`（`agent.go`）。Anthropic 支持在 tool result 中带图片，OpenAI 和 Google 不支持。Crush 的解决方案：

```go
func (a *sessionAgent) workaroundProviderMediaLimitations(messages []fantasy.Message, model Model) []fantasy.Message {
    providerSupportsMedia := model.ModelCfg.Provider == "anthropic" ||
        model.ModelCfg.Provider == "bedrock"
    if providerSupportsMedia {
        return messages  // Anthropic 原生支持，不用改
    }

    // 对其他 provider：把图片从 tool result 里抽出来，
    // 伪装成紧随其后的 user message
    for _, part := range msg.Content {
        if media := extractMedia(part); media != nil {
            // 替换 tool result 为文本占位符
            textParts = append(textParts, "[Image loaded - see attached]")
            // 注入一条 user message 带上图片
            mediaFiles = append(mediaFiles, media)
        }
    }
}
```

这不是 bug，这是**现实**。当你的工具系统同时支持的 `view` 返回图片时，这张图片在 Anthropic 上可以直接作为 tool result 回传，但在 OpenAI/Google 上必须被"偷渡"成 user message。模型看到的对话历史实际上被悄悄重写了。

类似的 provider 特殊处理还有很多：

- Anthropic 需要 `interleaved-thinking` beta header
- OpenAI 要区分 Chat Completions API 和 Responses API（`openai.IsResponsesModel()`）
- OpenRouter 支持 `:exacto` 后缀优化特定模型
- Google 的 thinking 配置格式和 Anthropic 完全不同
- `hyper` provider 根据模型名自动推断用哪个 SDK（含 `claude` 用 Anthropic，含 `gpt` 用 OpenAI）

**教训：多 provider 支持不是写几个 adapter 的事，而是在每一个边缘情况上反复打补丁。**

## 发现四：三层 Prompt Caching 策略

Crush 在 `agent.go` 的 `PrepareStep` 回调中实现了精细的 Anthropic cache control：

```go
PrepareStep: func(callContext context.Context, options fantasy.PrepareStepFunctionOptions) {
    // 第 1 层：最后一条 system message 加 cache control
    for i, msg := range prepared.Messages {
        if msg.Role == fantasy.MessageRoleSystem {
            lastSystemRoleInx = i
        } else if !systemMessageUpdated {
            prepared.Messages[lastSystemRoleInx].ProviderOptions = cacheControl
            systemMessageUpdated = true
        }
    }

    // 第 2 层：最后 2 条消息加 cache control
    if i > len(prepared.Messages)-3 {
        prepared.Messages[i].ProviderOptions = cacheControl
    }
}
```

加上之前在 `Run` 方法中的：

```go
// 第 3 层：最后一个 tool 定义加 cache control
agentTools[len(agentTools)-1].SetProviderOptions(a.getCacheControlOptions())
```

三层缓存：system prompt → tool 定义 → 最近的对话。这确保了：
- 不变的 system prompt 只发送一次
- 工具定义（可能很长）被缓存
- 最近的对话上下文在连续 tool loop 中被缓存

对比 Codex 的教训——他们曾因 MCP 工具排序不稳定导致整个 prompt 缓存失效。Crush 显式排序工具（`slices.SortFunc` 按名称），并在最后一个工具上打缓存标记，规避了这个问题。

## 发现五：Edit 工具的"文件时间锁"

Crush 的 edit 工具有一个大多数 agent 没有的安全机制：**修改前检查文件是否在上次读取之后被外部修改过**。

```go
// 必须先读过文件才能编辑
lastRead := edit.filetracker.LastReadTime(ctx, sessionID, filePath)
if lastRead.IsZero() {
    return "you must read the file before editing it. Use the View tool first"
}

// 检查文件的 mod time 是否比上次读取更新
modTime := fileInfo.ModTime().Truncate(time.Second)
if modTime.After(lastRead) {
    return fmt.Sprintf("file %s has been modified since it was last read", filePath)
}
```

同时，每次编辑都会在 SQLite 中记录文件的完整版本历史：

```go
// 如果用户手动改了文件（mod time 变了），记录中间版本
if file.Content != oldContent {
    edit.files.CreateVersion(ctx, sessionID, filePath, oldContent)
}
// 记录新版本
edit.files.CreateVersion(ctx, sessionID, filePath, newContent)
```

这意味着 Crush 有完整的编辑时间线。当 agent 改坏了文件，你可以回溯到任意一个版本——不是 git 级别的，而是每次编辑操作级别的。

## 发现六：Agentic Fetch — 用便宜模型做脏活

`agentic_fetch` 不是一个简单的"下载网页"工具，而是一个**完整的子 agent**，使用 **small model**（不是 large model）来分析网页内容：

```go
func (c *coordinator) agenticFetchTool(...) {
    // 关键：用 small model 做 web 内容分析
    _, small, _ := c.buildAgentModels(ctx, true)

    agent := NewSessionAgent(SessionAgentOptions{
        LargeModel: small,  // 注意：两个都是 small
        SmallModel: small,
        // ...
    })

    // 创建独立 session，自动批准所有权限
    c.permissions.AutoApproveSession(session.ID)
}
```

这个子 agent 有自己的独立临时目录、独立 session、自动批准的权限，以及 6 个工具（`web_fetch`、`web_search`、`glob`、`grep`、`sourcegraph`、`view`）。

大网页内容（超过阈值）会先写入临时文件，然后让子 agent 用 `view` 和 `grep` 工具去分析，而不是把整个页面塞进 prompt。

**成本逻辑**：网页分析不需要 Claude Opus 或 GPT-4o 级别的推理能力。用 Haiku 或 GPT-4o-mini 就够了，成本可能差 10-20 倍。这是一个很务实的工程决策。

## 发现七：392 行的 System Prompt

Crush 的 coder prompt 模板（`coder.md.tpl`）有 392 行，是我见过的最详细的 coding agent system prompt 之一。它用 XML 标签组织成 11 个模块：

```
<critical_rules>     — 13 条不可违反的核心规则
<communication_style> — 输出风格（<4 行、不用 emoji、一个词就回答）
<code_references>     — 引用代码时必须用 file:line 格式
<workflow>            — 每个任务的完整执行流程
<task_completion>     — 确保任务 100% 完成的检查清单
<error_handling>      — 遇到错误时的分步排查策略
<memory_instructions> — 何时更新记忆文件
<code_conventions>    — 编码规范（先看再改、匹配风格）
<testing>             — 改完代码必须跑测试
<tool_usage>          — 工具使用指南
<proactiveness>       — 自主性边界（做就做完，别问）
<final_answers>       — 回复长度规则
```

几个有趣的设计决策：

**"别问，去做"**：
> Don't ask questions — search, read, think, decide, act. Systematically try alternative strategies until the task is complete or you hit a hard external limit.

这和 Claude Code 的"不确定就问用户"形成鲜明对比。Crush 选择了极端自主化。

**Git 状态注入**：
prompt builder 在构建 system prompt 时自动执行 `git branch`、`git status --short`、`git log --oneline -n 3`，把结果注入到 prompt 的 `<env>` 段。agent 在开始对话时就知道当前分支、未提交的修改和最近的 commit。

**Context Files 通吃**：
```go
var defaultContextPaths = []string{
    ".github/copilot-instructions.md",
    ".cursorrules",
    "CLAUDE.md", "CLAUDE.local.md",
    "GEMINI.md",
    "crush.md", "AGENTS.md",
}
```

Crush 读取几乎所有竞品的项目指令文件。你为 Claude Code 写的 `CLAUDE.md`，Crush 也会读。这是一个聪明的策略：**不要求用户为 Crush 写专门的指令文件，直接复用他们已有的**。

## 发现八：安全模型的取舍

把所有发现串起来，Crush 的安全策略就清晰了：

| 层次 | Crush | Codex | Claude Code |
|------|-------|-------|-------------|
| **Shell** | 内嵌 POSIX 解释器 + AST 拦截 | 真实 shell + OS 沙箱 | 真实 bash |
| **命令过滤** | 60+ 禁止命令 + 子命令级过滤 | 沙箱内自由执行 | 权限提示 |
| **文件写入** | Permission prompt + mod time 检查 | 沙箱内自由写 | Permission prompt |
| **网络** | 禁止 curl/wget + 专用 fetch 工具 | 沙箱限制网络 | Permission prompt |
| **平台** | 8 个 OS | macOS + Linux | macOS + Linux + WSL |

Crush 没有 OS 级沙箱。这是它能支持 Android 和 BSD 的原因——不存在一个跨 8 个 OS 的统一沙箱方案。但它用 POSIX 解释器的 AST 拦截 + 精细的命令黑名单 + 子命令级参数过滤（比如 `npm install --global` 被禁但 `npm install` 被允许）来弥补。

这是一个明确的取舍：**用安全深度换平台广度**。

被禁止的命令分三类：
- **网络工具**：curl、wget、ssh、chrome、firefox 等 20 个
- **系统管理**：sudo、systemctl、crontab 等 11 个
- **包管理器**：apt、brew、pip、npm（全局安装）等 20+ 个

同时有一个"安全命令"白名单（`git status`、`git diff`、`ls`、`pwd` 等），这些命令不需要权限确认即可执行。

## 发现九：PostHog 遥测

Crush 内置了 PostHog 遥测，数据发送到 `data.charm.land`：

```go
const (
    endpoint = "https://data.charm.land"
    key      = "phc_4zt4VgDWLqbYnJYEwLRxFoaTL2noNrQij0C6E8k3I0V"
)

var baseProps = posthog.NewProperties().
    Set("GOOS", runtime.GOOS).
    Set("GOARCH", runtime.GOARCH).
    Set("TERM", os.Getenv("TERM")).
    Set("Version", version.Version)
```

收集的数据包括 OS、架构、终端类型、版本号、provider、model、token 用量、session 事件等。这在商业工具中很常见，但 Crush 使用的是 FSL-1.1-MIT 许可证（功能性源码许可证，不是传统开源）。结合遥测，Crush 更像是一个"源码可见的商业产品"而非社区开源项目。

## 总结：Crush 的工程哲学

读完代码，我认为 Crush 有一个一致的工程哲学：**在每个设计分叉点，选择"可移植性"而非"极致性能"或"极致安全"**。

- 内嵌 POSIX 解释器而非真实 shell → 可移植
- Permission prompt 而非 OS 沙箱 → 可移植
- Fantasy 多 provider 抽象而非单 provider 深度优化 → 可移植
- Catwalk 社区模型注册而非硬编码 → 可扩展
- 读取所有竞品的 context files → 兼容

这个哲学和 Charm 团队的 DNA 一致——他们从 2019 年就在做跨平台终端工具。Crush 不是一个"Go 版 Claude Code"，它是 Charm 生态自然延伸出的 AI 编码工具，继承了 Charm 对终端体验和跨平台的执念。

---

*本文基于 Crush v0.44.0（2026-02-22）的 263 个 Go 源文件分析。*
