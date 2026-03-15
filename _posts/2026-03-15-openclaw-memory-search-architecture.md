---
title: "深入解析 OpenClaw 的记忆搜索架构设计"
date: 2026-03-15 16:00:00 +0000
categories: [技术, AI]
tags: [openclaw, ai-agents, 记忆搜索, sqlite, 向量搜索]
description: "全面分析 OpenClaw 如何实现一个优雅的多层记忆搜索系统，从语义搜索到关键词匹配的平滑降级。"
toc: true
comments: true
---

# 深入解析 OpenClaw 的记忆搜索架构

全面分析 OpenClaw 如何实现一个优雅的多层记忆搜索系统，从语义搜索到关键词匹配的平滑降级。

## 引言

在构建需要记忆过去交互和积累知识的 AI Agent 时，我们面临一个基本挑战：如何平衡人类可读性、搜索性能和系统复杂度？OpenClaw 的记忆搜索系统提供了一个优雅的解决方案，对 AI Agent 架构设计有深远的影响。

通过逆向工程和源代码分析，我发现 OpenClaw 实现了一个精妙的三层记忆系统，能够从语义向量搜索无缝降级到关键词匹配，同时保持 Markdown 文件作为唯一的真相源。

## 架构概览

OpenClaw 的记忆搜索建立在三个核心原则之上：

1. **人类优先的数据格式**：所有记忆都存储为用户可以直接读写的 Markdown 文件
2. **渐进增强**：系统在多个能力级别上工作，从基础文件扫描到高级语义搜索
3. **零锁定**：索引纯粹用于加速——随时可以删除并从源文件重建

**系统架构层次：**

**第一层：用户接口**
- Markdown 文件（人类可读可写）

**第二层：SQLite 索引层**
- 元数据表 (files)
- 全文搜索索引 (FTS5)  
- 向量搜索索引 (vec)

**第三层：搜索运行时**
- 纯 FTS 模式
- 纯向量模式
- 混合搜索模式 (FTS + 向量)

## 三层设计

### 第一层：Markdown 文件（真相源）

基础设计简洁优雅：

- `~/.openclaw/workspace/`
  - `MEMORY.md` - 精选的长期记忆
  - `memory/` - 记忆文件夹
    - `2024-01-15.md` - 每日日志条目
    - `2024-01-16.md` - 每日日志条目
    - `projects.md` - 主题特定笔记

关键设计决策：
- **MEMORY.md**：保留重要的、精选的记忆（仅在私人会话中加载）
- **memory/\*.md**：日常日志和主题文件，用于运行上下文
- **纯文本**：无专有格式，可用任何文本编辑器
- **Git 友好**：每个更改都可以 diff 和追踪

### 第二层：SQLite 索引（性能加速器）

SQLite 数据库（`~/.openclaw/memory/main.sqlite`）作为一个可丢弃的加速层：

```sql
-- 核心架构（简化版）
CREATE TABLE chunks (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,           -- 源文件
    start_line INTEGER NOT NULL,  -- 文件中的位置
    end_line INTEGER NOT NULL,
    text TEXT NOT NULL,           -- 块内容
    embedding TEXT NOT NULL,      -- 向量（JSON 数组）
    updated_at INTEGER NOT NULL
);

-- 全文搜索索引
CREATE VIRTUAL TABLE chunks_fts USING fts5(
    text,                        -- 可搜索内容
    id UNINDEXED,
    path UNINDEXED,
    -- ... 其他元数据
);

-- 向量索引（当 sqlite-vec 可用时）
CREATE VIRTUAL TABLE chunks_vec USING vec0(
    chunk_id TEXT PRIMARY KEY,
    embedding FLOAT[1536]
);
```

这个索引的精妙之处在于它**不做**什么：
- 它从不修改源 Markdown 文件
- 它可以随时删除并重建
- 它不是基础功能所必需的

### 第三层：搜索运行时（多模式检索）

搜索层根据可用能力自适应：

```typescript
// 根据配置确定搜索模式
if (!this.provider) {
    // 模式1：纯 FTS（无 embedding provider）
    return this.searchFTSOnly(query);
} else if (!this.fts.available) {
    // 模式2：纯向量（FTS 不可用）
    return this.searchVectorOnly(query);
} else {
    // 模式3：混合（两者都可用）
    return this.searchHybrid(query);
}
```

## 存储层：Markdown 作为真相源

### 为什么选择 Markdown？

选择 Markdown 而非 JSON、YAML 或数据库是经过深思熟虑的：

1. **人类可编辑**：用户可以修正 AI 的错误、添加笔记、重新组织内容
2. **AI 原生**：LLM 无需特殊解析就能理解 Markdown 结构
3. **工具无关**：适用于任何文本编辑器、grep、git 等
4. **自文档化**：格式本身就提示了如何组织记忆

### 记忆文件约定

```markdown
# MEMORY.md 示例

## 关于用户
- 喜欢咖啡胜过茶
- 居住在北京
- 从事 AI 项目

## 重要决策
### 2024-01-15
决定使用 SQLite 实现向量搜索，而不是专用的向量数据库...
```

日常文件遵循类似的模式：

```markdown
# 2024-01-15

## 早晨讨论
用户询问了如何实现记忆搜索。要点：
- 需要平衡性能和简单性
- SQLite 似乎是个不错的折中方案
- 应该支持优雅降级
```

### 人机协作

OpenClaw 将记忆管理视为协作过程：
- AI 写入初始记忆
- 人类可以编辑、重组和纠正
- 文件监视器自动捕获更改
- 没有"同步冲突"——文件始终是权威的

## 索引层：SQLite 作为加速器

### 分块策略

文档被分割成重叠的块以实现最佳检索：

```typescript
const DEFAULT_CHUNK_TOKENS = 400;    // ~100-200 个词
const DEFAULT_CHUNK_OVERLAP = 80;    // ~20% 重叠
```

这确保了：
- 块足够大以保持上下文
- 重叠防止在边界处丢失信息
- 基于 token 的分割尊重句子边界

### 三个表，三个用途

1. **chunks**：存储可搜索的内容
   ```sql
   INSERT INTO chunks (id, path, start_line, end_line, text, embedding)
   VALUES (?, ?, ?, ?, ?, ?);
   ```

2. **chunks_fts**：全文搜索索引
   ```sql
   INSERT INTO chunks_fts (text, id, path, ...)
   SELECT text, id, path, ... FROM chunks;
   ```

3. **chunks_vec**：向量相似度索引（可选）
   ```sql
   INSERT INTO chunks_vec (chunk_id, embedding)
   SELECT id, embedding FROM chunks;
   ```

### 索引生命周期

索引自动维护：

```typescript
// 文件监视器触发重新索引
this.watcher = chokidar.watch(['MEMORY.md', 'memory/**/*.md'], {
    persistent: false,
    ignoreInitial: true,
});

this.watcher.on('change', () => {
    this.markDirty();
    this.scheduleSyncDebounced();
});
```

关键行为：
- 更改被去抖动（默认 1.5 秒）
- 同步异步运行以避免阻塞搜索
- 失败的同步不会破坏搜索（使用陈旧索引）

## 搜索层：混合检索

### 模式 1：纯 FTS（无 embedding）

当没有配置 embedding provider 时，系统回退到智能关键词搜索：

```typescript
// 从自然语言查询中提取关键词
const keywords = extractKeywords(query);
// "我之前告诉过你什么关于咖啡的事？" → ["告诉", "咖啡"]

// 独立搜索每个关键词
const resultSets = await Promise.all(
    keywords.map(term => this.searchKeyword(term))
);

// 合并并去重结果
const merged = mergeResultSets(resultSets);
```

关键词提取相当复杂：
- 删除停用词（"的"、"一个"、"什么"）
- 保留重要术语
- 处理多种语言

### 模式 2：纯向量（纯语义）

当只有 embedding 可用时：

```typescript
// 获取查询 embedding
const queryVec = await this.provider.embed(query);

// 使用余弦相似度查找相似块
const results = await this.searchVector(queryVec, limit);
```

向量搜索擅长：
- 语义相似性（"咖啡" ≈ "拿铁" ≈ "浓缩咖啡"）
- 语义改写（"我的机器" ≈ "我拥有的电脑"）
- 跨语言匹配（使用多语言模型）

### 模式 3：混合搜索（最佳点）

当 FTS 和向量搜索都可用时，OpenClaw 结合了它们的优势：

```typescript
// 并行运行两种搜索
const [vectorResults, keywordResults] = await Promise.all([
    this.searchVector(queryVec, candidateLimit),
    this.searchKeyword(query, candidateLimit)
]);

// 加权组合
const finalScore = 
    vectorWeight * vectorScore + 
    textWeight * textScore;
```

为什么混合搜索很重要：
- **向量**处理概念和改述
- **关键词**擅长精确匹配（ID、名称、代码）
- 两者结合覆盖更多检索场景

### 后处理管道

合并结果后，两个可选阶段优化输出：

#### 1. MMR（最大边际相关性）

通过平衡相关性和多样性来减少冗余：

```typescript
// MMR 评分
const mmrScore = λ * relevance - (1-λ) * maxSimilarityToSelected;
```

这防止返回五个几乎相同的关于同一主题的块。

#### 2. 时间衰减

提升最近的记忆而非旧的：

```typescript
// 基于年龄的指数衰减
const decayedScore = score * Math.exp(-λ * ageInDays);
```

默认半衰期为 30 天：
- 今天：100% 分数
- 1 周：~84% 分数
- 1 月：50% 分数
- 3 月：12.5% 分数

## 优雅降级

系统的降级策略确保它总是返回*某些东西*：

**降级流程：**

1. **Embedding API 失败** → 降级到 FTS 搜索
2. **SQLite 损坏** → 降级到文件扫描
3. **文件不可访问** → 返回空结果

每个级别提供逐渐基础但仍然功能性的搜索：

1. **完全混合**：语义 + 关键词搜索
2. **纯 FTS**：智能关键词提取和匹配
3. **文件扫描**：线性搜索 Markdown 文件（最后手段）

## 实现细节

### Embedding Provider

OpenClaw 支持多个 embedding provider，具有自动回退：

```typescript
// 提供者解析顺序
1. 本地（通过 llama.cpp 的 GGUF 模型）
2. OpenAI（text-embedding-3-small）
3. Google（gemini-embedding-001）
4. Voyage（voyage-3）
5. 无（纯 FTS 模式）
```

### 性能优化

1. **Embedding 缓存**
   ```sql
   CREATE TABLE embedding_cache (
       hash TEXT PRIMARY KEY,
       embedding TEXT,
       updated_at INTEGER
   );
   ```

2. **批处理**
   ```typescript
   // 用于大型语料库的 OpenAI 批处理 API
   if (settings.batch.enabled) {
       return this.batchEmbed(chunks);
   }
   ```

3. **SQLite-vec 扩展**
   ```sql
   -- 硬件加速的向量操作
   SELECT vec_distance_cosine(embedding, ?) AS distance
   FROM chunks_vec
   ORDER BY distance ASC
   ```

### 记忆作用域

不同的会话类型有不同的记忆访问权限：

```typescript
// 私人 DM 会话 - 完全访问
if (sessionType === 'direct') {
    return ['MEMORY.md', 'memory/**/*.md'];
}

// 群聊 - 有限访问
if (sessionType === 'group') {
    return ['memory/**/*.md']; // 没有 MEMORY.md
}
```

这防止了将个人上下文泄露到公共空间。

## 性能特征

基于实现，以下是预期性能：

### 索引性能
- **初始索引**：每 MB Markdown 约 1-2 秒
- **增量更新**：每个更改文件 <100ms
- **内存使用**：约为 Markdown 大小的 10-20 倍（含 embedding）

### 搜索性能
- **纯 FTS**：大多数查询 <10ms
- **向量搜索（使用 sqlite-vec）**：10k 块 <50ms
- **混合搜索**：典型 <100ms

### 可扩展性限制
- **测试达到**：100MB Markdown（约 50k 块）
- **实际限制**：1GB Markdown（取决于系统）
- **超过此限制**：考虑专用向量数据库

## 对 AI Agent 设计的启示

OpenClaw 的记忆系统教会了我们几个宝贵的经验：

### 1. 用户自主权很重要

通过将记忆保存在可编辑的 Markdown 中：
- 用户保持对其数据的控制
- 错误可以被纠正
- 组织可以自定义
- 无供应商锁定

### 2. 渐进增强有效

多层降级确保：
- 无需外部依赖的基本功能
- 资源允许时的增强能力
- 能力级别之间的平滑过渡
- 没有突然的功能丢失

### 3. 混合方法获胜

结合多种检索方法：
- 覆盖更多用例
- 提供回退选项
- 平衡优缺点
- 提高整体可靠性

### 4. 简单可扩展

使用 SQLite 而非专用向量数据库：
- 减少操作复杂性
- 实现本地优先操作
- 简化备份/恢复
- 降低入门门槛

## 结论

OpenClaw 的记忆搜索架构证明，复杂的 AI 能力不需要复杂的基础设施。通过深思熟虑地组合简单组件——Markdown 文件、SQLite 和可选嵌入——它实现了一个既强大又易于访问的系统。

关键见解是**最好的 AI 记忆系统是将人类和 AI 都视为一等参与者的系统**。Markdown 提供人机界面，SQLite 提供性能，降级策略确保它始终在某个级别上工作。

对于构建 AI Agent 的开发者，OpenClaw 的方法提供了一个蓝图：从人类可读的格式开始，添加可以重建的加速层，并始终提供优雅的降级路径。结果是用户可以信任、理解和控制的系统——这是处理个人信息的 AI 系统的基本品质。

## 参考文献和延伸阅读

- [OpenClaw 文档](https://docs.openclaw.ai/concepts/memory)
- [SQLite FTS5 扩展](https://sqlite.org/fts5.html)
- [理解 BM25](https://www.elastic.co/blog/understanding-bm25-ranking)
- [使用 SQLite 进行向量搜索](https://github.com/asg017/sqlite-vss)
- [混合搜索策略](https://www.pinecone.io/learn/hybrid-search/)

---

*本分析基于 OpenClaw 版本 2026.2.17 的源代码审查和运行时行为观察。*