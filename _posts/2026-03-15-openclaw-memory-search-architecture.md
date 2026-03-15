---
title: "Deep Dive into OpenClaw's Memory Search Architecture"
date: 2026-03-15 16:00:00 +0000
categories: [Technical, AI]
tags: [openclaw, ai-agents, memory-search, sqlite, vector-search]
description: "A comprehensive analysis of how OpenClaw implements an elegant, multi-tiered memory search system that gracefully degrades from semantic search to keyword matching."
author: duoqi
toc: true
comments: true
---

# Deep Dive into OpenClaw's Memory Search Architecture

> A comprehensive analysis of how OpenClaw implements an elegant, multi-tiered memory search system that gracefully degrades from semantic search to keyword matching.

## Table of Contents

1. [Introduction](#introduction)
2. [Architecture Overview](#architecture-overview)
3. [The Three-Layer Design](#the-three-layer-design)
4. [Storage Layer: Markdown as Truth](#storage-layer-markdown-as-truth)
5. [Index Layer: SQLite as Accelerator](#index-layer-sqlite-as-accelerator)
6. [Search Layer: Hybrid Retrieval](#search-layer-hybrid-retrieval)
7. [Graceful Degradation](#graceful-degradation)
8. [Implementation Details](#implementation-details)
9. [Performance Optimizations](#performance-optimizations)
10. [Lessons for AI Agent Design](#lessons-for-ai-agent-design)

## Introduction

When building AI agents that need to remember past interactions and accumulated knowledge, we face a fundamental challenge: how do we balance human readability, search performance, and system complexity? OpenClaw's memory search system offers an elegant solution that has profound implications for AI agent architecture.

Through reverse engineering and source code analysis, I discovered that OpenClaw implements a sophisticated three-tier memory system that seamlessly degrades from semantic vector search to keyword matching, all while keeping Markdown files as the single source of truth.

## Architecture Overview

OpenClaw's memory search is built on three core principles:

1. **Human-First Data Format**: All memories are stored as Markdown files that users can directly read and edit
2. **Progressive Enhancement**: The system works at multiple capability levels, from basic file scanning to advanced semantic search
3. **Zero Lock-in**: The index is purely for acceleration - delete it anytime and rebuild from source files

```
┌─────────────────────────────────────────────────┐
│                   User Interface                 │
│                  (Markdown Files)                │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│              SQLite Index Layer                  │
│  ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │   Metadata  │ │  Full-Text  │ │  Vector   │ │
│  │   (files)   │ │  Search     │ │  Search   │ │
│  └─────────────┘ └─────────────┘ └───────────┘ │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│               Search Runtime                     │
│  ┌─────────┐ ┌─────────┐ ┌─────────────────┐  │
│  │   FTS   │ │ Vector  │ │     Hybrid      │  │
│  │  Only   │ │  Only   │ │  (FTS+Vector)   │  │
│  └─────────┘ └─────────┘ └─────────────────┘  │
└─────────────────────────────────────────────────┘
```

## The Three-Layer Design

### Layer 1: Markdown Files (Source of Truth)

The foundation is beautifully simple:

```
~/.openclaw/workspace/
├── MEMORY.md           # Curated long-term memory
└── memory/
    ├── 2024-01-15.md   # Daily journal entries
    ├── 2024-01-16.md
    └── projects.md     # Topic-specific notes
```

Key design decisions:
- **MEMORY.md**: Reserved for important, curated memories (only loaded in private sessions)
- **memory/\*.md**: Daily logs and topic files for running context
- **Plain text**: No proprietary format, works with any text editor
- **Git-friendly**: Every change is diffable and trackable

### Layer 2: SQLite Index (Performance Accelerator)

The SQLite database (`~/.openclaw/memory/main.sqlite`) serves as a disposable acceleration layer:

```sql
-- Core schema (simplified)
CREATE TABLE chunks (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,           -- Source file
    start_line INTEGER NOT NULL,  -- Location in file
    end_line INTEGER NOT NULL,
    text TEXT NOT NULL,           -- Chunk content
    embedding TEXT NOT NULL,      -- Vector (JSON array)
    updated_at INTEGER NOT NULL
);

-- Full-text search index
CREATE VIRTUAL TABLE chunks_fts USING fts5(
    text,                        -- Searchable content
    id UNINDEXED,
    path UNINDEXED,
    -- ... other metadata
);

-- Vector index (when sqlite-vec available)
CREATE VIRTUAL TABLE chunks_vec USING vec0(
    chunk_id TEXT PRIMARY KEY,
    embedding FLOAT[1536]
);
```

The genius is in what this index **doesn't** do:
- It never modifies the source Markdown files
- It can be deleted and rebuilt anytime
- It's not required for basic functionality

### Layer 3: Search Runtime (Multi-Mode Retrieval)

The search layer adapts to available capabilities:

```typescript
// Determine search mode based on configuration
if (!this.provider) {
    // Mode 1: FTS-only (no embedding provider)
    return this.searchFTSOnly(query);
} else if (!this.fts.available) {
    // Mode 2: Vector-only (FTS unavailable)
    return this.searchVectorOnly(query);
} else {
    // Mode 3: Hybrid (both available)
    return this.searchHybrid(query);
}
```

## Storage Layer: Markdown as Truth

### Why Markdown?

The choice of Markdown over JSON, YAML, or a database is deliberate:

1. **Human Editable**: Users can fix AI mistakes, add notes, reorganize content
2. **AI Native**: LLMs understand Markdown structure without special parsing
3. **Tool Agnostic**: Works with any text editor, grep, git, etc.
4. **Self-Documenting**: The format itself suggests how to organize memories

### Memory File Conventions

```markdown
# MEMORY.md Example

## About the User
- Prefers coffee over tea
- Lives in Beijing
- Works on AI projects

## Important Decisions
### 2024-01-15
Decided to implement vector search using SQLite instead of a dedicated vector database...
```

Daily files follow a similar pattern:

```markdown
# 2024-01-15

## Morning Discussion
User asked about implementing memory search. Key points:
- Need to balance performance and simplicity
- SQLite seems like a good compromise
- Should support graceful degradation
```

### The Human in the Loop

OpenClaw treats memory curation as a collaborative process:
- The AI writes initial memories
- Humans can edit, reorganize, and correct
- Changes are picked up automatically by file watchers
- No "sync conflicts" - the file is always authoritative

## Index Layer: SQLite as Accelerator

### Chunking Strategy

Documents are split into overlapping chunks for optimal retrieval:

```typescript
const DEFAULT_CHUNK_TOKENS = 400;    // ~100-200 words
const DEFAULT_CHUNK_OVERLAP = 80;    // ~20% overlap
```

This ensures:
- Chunks are large enough to maintain context
- Overlap prevents losing information at boundaries
- Token-based splitting respects sentence boundaries

### Three Tables, Three Purposes

1. **chunks**: Stores the searchable content
   ```sql
   INSERT INTO chunks (id, path, start_line, end_line, text, embedding)
   VALUES (?, ?, ?, ?, ?, ?);
   ```

2. **chunks_fts**: Full-text search index
   ```sql
   INSERT INTO chunks_fts (text, id, path, ...)
   SELECT text, id, path, ... FROM chunks;
   ```

3. **chunks_vec**: Vector similarity index (optional)
   ```sql
   INSERT INTO chunks_vec (chunk_id, embedding)
   SELECT id, embedding FROM chunks;
   ```

### Index Lifecycle

The index is automatically maintained:

```typescript
// File watcher triggers reindex
this.watcher = chokidar.watch(['MEMORY.md', 'memory/**/*.md'], {
    persistent: false,
    ignoreInitial: true,
});

this.watcher.on('change', () => {
    this.markDirty();
    this.scheduleSyncDebounced();
});
```

Key behaviors:
- Changes are debounced (1.5 seconds default)
- Sync runs asynchronously to avoid blocking search
- Failed syncs don't break search (uses stale index)

## Search Layer: Hybrid Retrieval

### Mode 1: FTS-Only (No Embeddings)

When no embedding provider is configured, the system falls back to intelligent keyword search:

```typescript
// Extract keywords from natural language query
const keywords = extractKeywords(query);
// "What did I tell you about coffee?" → ["tell", "coffee"]

// Search each keyword independently
const resultSets = await Promise.all(
    keywords.map(term => this.searchKeyword(term))
);

// Merge and deduplicate results
const merged = mergeResultSets(resultSets);
```

The keyword extraction is surprisingly sophisticated:
- Removes stop words ("the", "a", "what")
- Preserves important terms
- Handles multiple languages

### Mode 2: Vector-Only (Pure Semantic)

When only embeddings are available:

```typescript
// Get query embedding
const queryVec = await this.provider.embed(query);

// Find similar chunks using cosine similarity
const results = await this.searchVector(queryVec, limit);
```

Vector search excels at:
- Semantic similarity ("coffee" ≈ "latte" ≈ "espresso")
- Paraphrasing ("my machine" ≈ "the computer I own")
- Cross-lingual matching (with multilingual models)

### Mode 3: Hybrid Search (The Sweet Spot)

When both FTS and vector search are available, OpenClaw combines their strengths:

```typescript
// Run both searches in parallel
const [vectorResults, keywordResults] = await Promise.all([
    this.searchVector(queryVec, candidateLimit),
    this.searchKeyword(query, candidateLimit)
]);

// Weighted combination
const finalScore = 
    vectorWeight * vectorScore + 
    textWeight * textScore;
```

Why hybrid search matters:
- **Vectors** handle concepts and paraphrasing
- **Keywords** excel at exact matches (IDs, names, code)
- Together they cover more retrieval scenarios

### Post-Processing Pipeline

After merging results, two optional stages refine the output:

#### 1. MMR (Maximum Marginal Relevance)

Reduces redundancy by balancing relevance with diversity:

```typescript
// MMR scoring
const mmrScore = λ * relevance - (1-λ) * maxSimilarityToSelected;
```

This prevents returning five nearly-identical chunks about the same topic.

#### 2. Temporal Decay

Boosts recent memories over old ones:

```typescript
// Exponential decay based on age
const decayedScore = score * Math.exp(-λ * ageInDays);
```

With default half-life of 30 days:
- Today: 100% score
- 1 week: ~84% score  
- 1 month: 50% score
- 3 months: 12.5% score

## Graceful Degradation

The system's degradation strategy ensures it always returns *something*:

```
┌─────────────────┐
│ Embedding API   │ ──❌──┐
└─────────────────┘       │
                         ▼
┌─────────────────┐     ┌─────────────────┐
│ Vector Search   │ ──▶ │  FTS Fallback   │
└─────────────────┘     └─────────────────┘
                                │
                         ❌ SQLite corrupted
                                │
                                ▼
                         ┌─────────────────┐
                         │  File Scanning  │
                         └─────────────────┘
```

Each level provides progressively basic but still functional search:

1. **Full Hybrid**: Semantic + keyword search
2. **FTS-Only**: Intelligent keyword extraction and matching
3. **File Scan**: Linear search through Markdown files (last resort)

## Implementation Details

### Embedding Providers

OpenClaw supports multiple embedding providers with automatic fallback:

```typescript
// Provider resolution order
1. Local (GGUF model via llama.cpp)
2. OpenAI (text-embedding-3-small)
3. Google (gemini-embedding-001)
4. Voyage (voyage-3)
5. None (FTS-only mode)
```

### Performance Optimizations

1. **Embedding Cache**
   ```sql
   CREATE TABLE embedding_cache (
       hash TEXT PRIMARY KEY,
       embedding TEXT,
       updated_at INTEGER
   );
   ```

2. **Batch Processing**
   ```typescript
   // OpenAI Batch API for large corpus
   if (settings.batch.enabled) {
       return this.batchEmbed(chunks);
   }
   ```

3. **SQLite-vec Extension**
   ```sql
   -- Hardware-accelerated vector operations
   SELECT vec_distance_cosine(embedding, ?) AS distance
   FROM chunks_vec
   ORDER BY distance ASC
   ```

### Memory Scoping

Different session types have different memory access:

```typescript
// Private DM session - full access
if (sessionType === 'direct') {
    return ['MEMORY.md', 'memory/**/*.md'];
}

// Group chat - limited access
if (sessionType === 'group') {
    return ['memory/**/*.md']; // No MEMORY.md
}
```

This prevents leaking personal context into public spaces.

## Performance Characteristics

Based on the implementation, here's what to expect:

### Indexing Performance
- **Initial index**: ~1-2 seconds per MB of Markdown
- **Incremental updates**: <100ms per changed file
- **Memory usage**: ~10-20x the Markdown size (with embeddings)

### Search Performance
- **FTS-only**: <10ms for most queries
- **Vector search (with sqlite-vec)**: <50ms for 10k chunks
- **Hybrid search**: <100ms typical

### Scalability Limits
- **Tested up to**: 100MB of Markdown (~50k chunks)
- **Practical limit**: 1GB of Markdown (system-dependent)
- **Beyond that**: Consider dedicated vector database

## Lessons for AI Agent Design

OpenClaw's memory system teaches several valuable lessons:

### 1. User Agency Matters

By keeping memories in editable Markdown:
- Users maintain control over their data
- Mistakes can be corrected
- Organization can be customized
- No vendor lock-in

### 2. Progressive Enhancement Works

The multi-tier degradation ensures:
- Basic functionality without external dependencies
- Enhanced capabilities when resources allow
- Smooth transitions between capability levels
- No sudden feature loss

### 3. Hybrid Approaches Win

Combining multiple retrieval methods:
- Covers more use cases
- Provides fallback options
- Balances strengths and weaknesses
- Improves overall reliability

### 4. Simplicity Scales

Using SQLite instead of a specialized vector database:
- Reduces operational complexity
- Enables local-first operation
- Simplifies backup/restore
- Lowers barrier to entry

## Conclusion

OpenClaw's memory search architecture demonstrates that sophisticated AI capabilities don't require complex infrastructure. By thoughtfully combining simple components - Markdown files, SQLite, and optional embeddings - it achieves a system that is both powerful and accessible.

The key insight is that **the best AI memory system is one that treats both humans and AI as first-class participants**. Markdown provides the human interface, SQLite provides the performance, and the degradation strategy ensures it always works at some level.

For developers building AI agents, OpenClaw's approach offers a blueprint: start with human-readable formats, add acceleration layers that can be rebuilt, and always provide graceful degradation paths. The result is a system that users can trust, understand, and control - essential qualities for AI systems that handle personal information.

## References and Further Reading

- [OpenClaw Documentation](https://docs.openclaw.ai/concepts/memory)
- [SQLite FTS5 Extension](https://sqlite.org/fts5.html)
- [Understanding BM25](https://www.elastic.co/blog/understanding-bm25-ranking)
- [Vector Search with SQLite](https://github.com/asg017/sqlite-vss)
- [Hybrid Search Strategies](https://www.pinecone.io/learn/hybrid-search/)

---

*This analysis is based on OpenClaw version 2026.2.17 source code examination and runtime behavior observation.*