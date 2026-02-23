---
title: "Coding Agent 的上下文压缩：6 个项目的实现对比"
date: 2026-02-23 05:00:00 +0800
categories: [AI]
tags: [coding-agent, compaction, context-window, claude-code, codex, crush, openclaw]
mermaid: true
---

> Agent 做的事越多，上下文就越长。当对话撞上上下文窗口天花板时，每个 coding agent 都必须回答同一个问题：**丢掉什么，留下什么？** 读完 6 个项目的实现后，我发现这个看似简单的问题，背后藏着截然不同的工程决策。

## 为什么需要 Compaction

Coding agent 和普通聊天机器人有一个根本区别：**工具循环会指数级膨胀上下文**。

一个"帮我修 bug"的请求，agent 可能需要：读 5 个文件（每个几百行）→ 跑测试（输出几千行）→ 改 3 个文件 → 再跑测试 → 再改。每一步的工具调用和返回结果都要保留在对话里，因为 LLM 需要看到完整的执行历史才能做出下一步决策。

一个中等复杂度的编码任务，轻松消耗 50K-100K tokens。而真实的开发场景往往是连续工作数小时，对话长度可以达到 200K+ tokens。即使是 200K 窗口的模型，也撑不住。

所有 coding agent 都必须解决这个问题。但它们的解决方案大相径庭。

## 什么时候压缩：触发机制

### 阈值设计

每个项目都要回答一个问题：**上下文用到多少比例时开始压缩？**

| 项目 | 自动触发阈值 | 手动命令 | 可配置？ |
|------|-------------|---------|---------|
| Claude Code | ~83.5% 有效窗口 | `/compact` | 是（环境变量） |
| Codex | 90% 上下文窗口 | `/compact` | 是（per-model） |
| Crush | 剩余 ≤ 20K 或 ≤ 20% | 无 | 可禁用 |
| OpenClaw | 超过上下文窗口 - 20K | 无 | 是 |
| Aider | 超过 chat history token budget | 无（全自动） | 是 |
| Cline | ~80% | Auto Compact 开关 | 开关 |

这些阈值反映了不同的风险偏好。Codex 用 90% 是因为它有 WebSocket sticky routing 带来的高效 prompt caching——可以尽量晚压缩。Crush 对大模型（>200K）留 20K 绝对缓冲，对小模型留 20% 相对缓冲——这个双阈值策略比固定百分比更适应不同模型。

**Crush 的双阈值策略值得展开**：

```go
if cw > largeContextWindowThreshold {  // > 200K
    threshold = largeContextWindowBuffer    // 固定 20K
} else {
    threshold = int64(float64(cw) * smallContextWindowRatio)  // 20%
}
```

为什么不统一用百分比？因为 200K 的 20% 是 40K——留 40K 的缓冲太浪费了，足够多做好几轮工具调用。而 4K 模型的 20% 只有 800 tokens，已经是最低限度。固定 20K 对大模型更经济，20% 对小模型更安全。

### Claude Code 的可配置阈值

Claude Code 是唯一支持用户自定义触发阈值的项目：

```bash
export CLAUDE_AUTOCOMPACT_PCT_OVERRIDE=90  # 推迟到 90% 再压缩
```

还支持在 settings 中完全关闭：

```json
{ "autoCompact": false }
```

以及自定义压缩指令——你可以告诉它压缩时重点保留什么：

```
/compact only keep the names of the websites we reviewed
/compact preserve the coding patterns we established
```

这种灵活性在其他项目中看不到。

## 怎么压缩：三种策略

读完 6 个项目的源码，我发现压缩策略可以分为三类。

### 策略一：LLM 自己总结自己（主流）

Claude Code、Codex、Crush、OpenClaw 都用这个方法：**把对话历史发给 LLM，让它生成摘要，用摘要替换原始历史**。

但细节差异很大。

**Claude Code** 要求 LLM 生成 9 个结构化段落：

1. 主要请求和意图
2. 关键技术概念
3. 文件和代码段（含完整代码片段）
4. 错误和修复
5. 问题解决过程
6. **所有用户消息（逐字保留）**
7. 待完成任务
8. 当前工作
9. 下一步建议（含对话原文引用）

这是 6 个项目中**最详细的摘要模板**——1121 tokens 的 prompt 专门指导如何生成摘要。它甚至要求"逐字保留所有用户消息"，确保用户的原始意图不会在压缩中丢失。

**Codex** 的摘要 prompt 最精简——只有 4 条要求：

```
You are performing a CONTEXT CHECKPOINT COMPACTION.
Create a handoff summary for another LLM that will resume the task.

Include:
- Current progress and key decisions made
- Important context, constraints, or user preferences
- What remains to be done (clear next steps)
- Any critical data, examples, or references needed to continue
```

但它有一个独特的**前缀机制**——摘要注入时会加上一段"交接说明"：

```
Another language model started to solve this problem and produced a summary
of its thinking process. You also have access to the state of the tools that
were used by that language model. Use this to build on the work that has already
been done and avoid duplicating work.
```

把压缩后的续接框定为"接手别人的工作"，而不是"回忆自己做过的事"。这个心智模型的差异可能影响 LLM 的续接质量。

**Crush** 的摘要模板最长（392 行中有专门的 summary 模板），要求包含 5 个部分：Current State、Files & Changes、Technical Context、Strategy & Approach、Exact Next Steps。它特别强调"**写得像在给接手同事做交接**"，并且明确说"长度不限，宁多勿少"。

还有一个独特设计：如果 session 有 Todo 列表，会动态注入到摘要 prompt 中：

```go
func buildSummaryPrompt(todos []session.Todo) string {
    sb.WriteString("Provide a detailed summary of our conversation above.")
    if len(todos) > 0 {
        sb.WriteString("\n\n## Current Todo List\n\n")
        for _, t := range todos {
            fmt.Fprintf(&sb, "- [%s] %s\n", t.Status, t.Content)
        }
    }
    return sb.String()
}
```

**OpenClaw** 用分阶段摘要——把消息分成多个块，分别总结，最后合并。它还追踪工具失败记录和文件操作记录，注入到最终摘要中。这在其他项目中看不到。

### 策略二：递归分治（Aider）

Aider 不是一次性总结整个对话，而是用递归的 split-and-summarize：

```python
def summarize_real(self, messages, depth=0):
    # 在 token budget 的一半位置切分
    # 确保切分点在 assistant 消息结束处（保持对话完整性）
    head = messages[:split_index]
    tail = messages[split_index:]

    summary = self.summarize_all(head)  # 用 LLM 总结前半部分

    if summary_tokens + tail_tokens < self.max_tokens:
        return summary + tail  # 够了，返回

    # 还是太长？递归（最多 3 层）
    return self.summarize_real(summary + tail, depth + 1)
```

这种方法的好处是**最近的消息永远原封不动保留**，只有越老的消息被压缩得越狠——符合直觉，因为最近的操作最相关。

Aider 的摘要 prompt 也很有意思——它要求 LLM 用**第一人称**写，以用户的口吻：

```python
summarize = """...
Phrase the summary with the USER in first person,
telling the ASSISTANT about the conversation.
Write *as* the user.
Start the summary with "I asked you...".
"""
```

### 策略三：观测遮罩（JetBrains 研究）

JetBrains 在 2025 年 12 月发表的研究提出了一个不同的思路：**不总结，直接遮罩**。

在 SWE-bench Verified（500 个实例）上，他们对比了两种方法：

- **观测遮罩**：保留 agent 的推理和工具调用，但用占位符替换旧的工具返回结果
- **LLM 摘要**：用另一个模型压缩旧的对话

结果：观测遮罩在 4/5 的测试设定中表现更好。LLM 摘要虽然也能减少 50% 成本，但导致 agent **多运行 13-15%**——因为摘要模糊了"应该停止"的信号。

这个发现挑战了主流的"用 LLM 总结"策略。目前还没有主流 coding agent 采用纯观测遮罩，但它指出了一个方向：**工具返回结果占据了上下文的大部分空间，而 agent 的推理和工具调用本身很小**。只遮罩结果、保留推理，可能是更好的策略。

## 压缩后怎么注入回去

这是一个容易被忽视但极其关键的设计决策。

### Codex：双模式注入

Codex 区分两种场景：

- **手动/预回合压缩**（`DoNotInject`）：清除 `reference_context_item`，下一轮自然重新注入完整的初始上下文（AGENTS.md、环境信息等）
- **工具循环中段压缩**（`BeforeLastUserMessage`）：初始上下文必须插入到最后一条用户消息之前——因为模型训练时看到的压缩摘要总是在历史最后

```rust
// 优先级：
// 1. 最后一条真实用户消息之前（首选）
// 2. 摘要消息之前（降级）
// 3. 最后一个 compaction item 之前（最终降级）
// 4. 追加到末尾（兜底）
let insertion_index = last_real_user_index
    .or(last_user_or_summary_index)
    .or(last_compaction_index);
```

这个四级降级策略说明 Codex 在这个问题上踩过坑——不是所有情况下都能找到理想的注入点。

### Crush：Assistant → User 的角色转换

Crush 的注入方式最巧妙。摘要生成时，它被存为 `Assistant` 消息（因为是 LLM 生成的）。但下次加载 session 时：

```go
func (a *sessionAgent) getSessionMessages(ctx context.Context, session session.Session) ([]message.Message, error) {
    msgs, err := a.messages.List(ctx, session.ID)
    if session.SummaryMessageID != "" {
        summaryMsgIndex := -1
        for i, msg := range msgs {
            if msg.ID == session.SummaryMessageID {
                summaryMsgIndex = i
                break
            }
        }
        if summaryMsgIndex != -1 {
            msgs = msgs[summaryMsgIndex:]    // 丢弃摘要之前的所有消息
            msgs[0].Role = message.User       // 把摘要的角色从 Assistant 改成 User！
        }
    }
    return msgs, nil
}
```

**为什么要改角色？** 因为 LLM 的对话格式要求消息交替出现（User → Assistant → User → ...）。如果摘要保持 Assistant 角色，它后面不能直接跟 Assistant 消息。改成 User 之后，摘要变成了"用户提供的上下文"，之后可以正常续接。

同时，`PromptTokens` 被重置为 0——摘要成为新的上下文起点。

### Claude Code：Compaction Block

Claude Code 用 API 级别的 `compaction` 类型标记：

```json
{
  "content": [
    { "type": "compaction", "content": "Summary of the conversation..." },
    { "type": "text", "text": "Based on our conversation so far..." }
  ]
}
```

API 自动丢弃 compaction block 之前的所有消息。多次压缩时，只有最后一个 compaction block 是有效的。

对于压缩前读过但内容太大的文件，会注入一条引用：

```
Note: ${filename} was read before the last conversation was summarized,
but the contents are too large to include. Use Read tool if you need to access it.
```

这样 agent 知道自己之前读过这个文件，但需要重新读取。

### Codex：保留用户消息

Codex 在替换历史时，不是只留摘要——它保留了**最近 20,000 tokens 的用户消息**（按时间倒序选取）：

```rust
const COMPACT_USER_MESSAGE_MAX_TOKENS: usize = 20_000;

for message in user_messages.iter().rev() {
    if remaining == 0 { break; }
    let tokens = approx_token_count(message);
    if tokens <= remaining {
        selected_messages.push(message.clone());
        remaining = remaining.saturating_sub(tokens);
    } else {
        let truncated = truncate_text(message, TruncationPolicy::Tokens(remaining));
        selected_messages.push(truncated);
        break;
    }
}
```

压缩后的历史结构是：`initial_context + 近期用户消息 + 摘要`。这确保 LLM 能看到用户最近说了什么，而不仅仅依赖摘要的转述。

## 工具结果截断：被忽视的第一道防线

在触发 LLM 摘要之前，还有一道更基础的防线：**截断过大的单次工具返回结果**。一个 `find / -name "*.log"` 的输出可能有几十万字符，不截断的话一条工具结果就能吃掉大半个上下文窗口。

| 项目 | 单次工具结果上限 | 截断方式 |
|------|----------------|---------|
| Claude Code | 动态（基于窗口） | 保留头尾 |
| Codex | 1MiB | 保留头尾 |
| Crush | 30,000 字符 | 保留前后各一半 |
| OpenClaw | 上下文窗口的 30%（最大 400K 字符） | 按比例分配 + 换行符对齐 |

OpenClaw 的方案最精细——它按 block 的原始比例分配截断预算，并且在截断点寻找最近的换行符（80% 以内），避免在代码行中间断开。截断后附加警告：

```
⚠️ Content truncated — original was too large for the model's context window.
The content above is a partial view. If you need more, request specific sections
or use offset/limit parameters to read smaller chunks.
```

## 压缩失败时怎么办

压缩本身也可能失败——摘要 prompt 加上完整对话历史可能超过上下文窗口（尤其是用小模型压缩长对话时）。各项目的容错策略：

**Codex** 最完善——如果压缩时上下文溢出，它从历史开头逐条删除，直到 prompt 能放进窗口。同时有指数退避重试：

```rust
Err(e @ CodexErr::ContextWindowExceeded) => {
    if turn_input_len > 1 {
        history.remove_first_item();  // 从开头删
        truncated_count += 1;
        retries = 0;
        continue;
    }
}
```

**OpenClaw** 最多重试 3 次。如果 LLM 摘要失败，降级到工具结果截断。如果截断也解决不了，放弃并报错：

```typescript
if (overflowCompactionAttempts < MAX_OVERFLOW_COMPACTION_ATTEMPTS) {
    // 尝试 LLM 压缩
}
// 降级到工具结果截断
if (!toolResultTruncationAttempted) {
    const truncResult = await truncateOversizedToolResultsInSession({...});
    if (truncResult.truncated) {
        overflowCompactionAttempts = 0;  // 重置计数器
        continue;
    }
}
// 最终放弃
return { text: "Context overflow: prompt too large for the model...", isError: true };
```

**Crush** 最简洁——压缩失败就失败，agent 停止当前任务。但如果 agent 在压缩前有未完成的工具调用，它会把原始请求重新入队，并修改 prompt 标注这是被中断的：

```go
call.Prompt = fmt.Sprintf(
    "The previous session was interrupted because it got too long, "+
    "the initial user request was: `%s`", call.Prompt)
```

**Claude Code** 的压缩错误被静默吞掉——session 继续运行，用户可以手动 `/compact` 重试。

## 谁来做压缩：模型选择

一个有趣的决策是：**用哪个模型来生成摘要？**

| 项目 | 摘要模型 | 原因 |
|------|---------|------|
| Claude Code | 同一个模型 | 质量优先，保持一致性 |
| Codex | 同一个模型（OpenAI only）或远程 API | OpenAI 有专门的 compact API |
| Crush | 大模型（largeModel） | 质量优先 |
| Cursor | 更小的 "flash" 模型 | 速度和成本优先 |
| Aider | 可配置（支持 fallback） | 灵活性优先 |
| OpenClaw | 由 pi-coding-agent 决定 | 委托给底层库 |

Cursor 是唯一明确使用更小模型做摘要的项目。这是一个成本 vs 质量的取舍——用 Haiku 做摘要可能便宜 10 倍，但摘要质量可能影响后续任务的准确性。

Codex 有一个独特设计：对 OpenAI 模型使用**远程压缩 API**（`compact_conversation_history`），非 OpenAI 模型用本地压缩。远程压缩让 OpenAI 可以在服务端优化压缩过程，比如利用已有的 prompt cache。

## 与 Prompt Caching 的交互

压缩和 prompt caching 是一对矛盾——压缩会改变对话历史，导致之前缓存的 prompt prefix 失效。各项目的处理方式：

**Codex** 用 `prompt_cache_key`（conversation ID）维持压缩前后的缓存连续性。压缩不改变 conversation ID，所以 API 端的缓存仍然有效。

**Crush** 在压缩后重置 `PromptTokens = 0`——摘要成为新的 prompt 起点，之前的缓存自然失效。但它的三层 cache control（system prompt + 最后一个 tool 定义 + 最近 2 条消息）在新一轮对话中会重新建立。

**Claude Code** 在 system prompt 上设置了独立的 `cache_control` 断点，确保系统提示的缓存不受对话压缩影响。只有对话部分的缓存会失效。

压缩后每次 LLM 调用都要重新建立缓存（cache miss），这是压缩的隐性成本。这也是为什么所有项目都尽量推迟压缩——越晚压缩，利用缓存的时间越长。

## 经验总结

### 压缩是架构的一部分，不是事后补丁

6 个项目都在早期就实现了压缩机制。Codex 专门写了 1107 行的 `compact.rs` 和独立的 `compact_remote.rs`。这不是"nice to have"——没有压缩的 coding agent 无法完成超过 30 分钟的任务。

### 先截断工具结果，再做 LLM 摘要

OpenClaw 的分层防御最清晰：工具结果截断（便宜、快）→ LLM 摘要（贵、慢）→ 放弃。大部分上下文膨胀来自工具返回结果——先处理它们，可以大幅延迟或避免 LLM 摘要。

### 保留用户原始消息比摘要更可靠

Codex 保留最近 20K tokens 的用户消息，Claude Code 要求摘要"逐字保留所有用户消息"。原因很简单：LLM 的摘要可能丢失微妙但重要的用户意图。用户说"不要改 API 接口"这种约束条件，在摘要中可能被弱化或遗漏。

### 压缩会降低准确性

Codex 在每次压缩后显示警告："Long threads and multiple compactions can cause the model to be less accurate. Start a new thread when possible."

这不是谦虚——JetBrains 的研究证实，LLM 摘要会导致 agent 多运行 13-15%（因为摘要模糊了停止信号）。压缩是一个必要的妥协，不是无损操作。

### 触发阈值没有银弹

太早压缩浪费准确性，太晚压缩有溢出风险。Crush 的双阈值策略（大模型固定缓冲、小模型百分比缓冲）是一个务实的折中。Claude Code 让用户自己调是另一种思路——承认没有最优值，把选择权交给用户。

---

*基于 Claude Code（2026-02）、Codex（3913 commits，compact.rs 1107 行）、Crush v0.44.0、OpenClaw v2026.2.18、Aider（history.py）、Cline 源码分析，以及 JetBrains 2025 年 12 月发表的 SWE-bench 研究。*
