---
title: "【工具】AI Coding 配置管理工具：CC-Switch / Antigravity / zcf / CCR / CRS"
date: 2026-02-28 20:00:00 +0800
categories: [Tools]
tags: [claude-code, codex, gemini, mcp, tools]
---

> Claude Code、Codex、Gemini CLI 火了，配套的配置管理工具也涌现出来。CC-Switch、Antigravity-Manager、zcf、CCR、CRS 都解决类似的问题——API 配置管理、多账号切换、中转服务集成。本文介绍这几款工具。

## 为什么需要这些工具？

如果你使用 Claude Code、Codex、Gemini CLI，会遇到这些问题：

1. **Provider 切换麻烦** — 官方 API、中转服务、自定义 endpoint，要手动改配置文件
2. **多账号管理** — 不同账号、不同额度
3. **成本控制** — 官方贵，中转便宜
4. **配置分散** — 每个工具都有自己的配置文件

这些工具就是为了解决这些问题。

---

## 工具对比

| 工具 | Stars | 定位 | 特点 |
|------|-------|------|------|
| **CC-Switch** | 21.9k | 一站式配置管理 | 功能最全 |
| **Antigravity-Manager** | 24.8k | 协议代理 + 账号管理 | Web Session → API |
| **zcf** | 5.5k | 零配置快速启动 | 新手友好 |
| **CCR** | ~257 | Claude Code Router | 轻量级代理 |
| **CRS** | - | 自建中转服务 | 拼车共享 |

---

## CC-Switch

**GitHub**: https://github.com/farion1231/cc-switch

All-in-One assistant tool for Claude Code, Codex & Gemini CLI。

**核心功能**:
- Provider 管理（一键切换 API 配置）
- MCP Server 统一管理
- Skills 管理（从 GitHub 自动扫描安装）
- Prompts 管理（系统提示词预设）
- 配置导入/导出/备份

**安装 (macOS)**:
```bash
brew tap farion1231/ccswitch
brew install --cask cc-switch
```

**特点**: 功能最全面，有 GUI 和 CLI 两个版本。

---

## Antigravity-Manager

**GitHub**: https://github.com/lbjlaq/Antigravity-Manager

专业级 AI 账号管理与协议代理系统。

**核心功能**:
- 多账号管理
- 协议转换（Web Session → API）
- 智能请求调度
- 负载均衡

**特点**: 将 Google/Anthropic 的 Web Session 转为标准 API，支持多账号负载均衡。

**适合**: 需要高性能 API 中转的用户。

---

## zcf

**GitHub**: https://github.com/UfoMiao/zcf

Zero-Config Code Flow，零配置快速启动。

**安装**:
```bash
npx zcf i  # 全自动安装
```

**核心功能**:
- 一键安装配置
- workflows 自动生成
- API/CCR 快速接入
- MCP 自动配置

**特点**: 强调"零配置"，新手友好。

---

## CCR

**GitHub**: https://github.com/musistudio/claude-code-router

Claude Code Router，轻量级代理。

**核心功能**:
- 多 provider 路由
- API 代理
- 请求转发

**特点**: 专注于路由功能，轻量级。

**适合**: 只需要简单代理的开发者。

---

## CRS

**GitHub**: https://github.com/Wei-Shaw/claude-relay-service

自建 Claude Code 镜像/中转服务，支持拼车共享。

**核心功能**:
- 多账户管理 + 自动轮换
- 给每个用户分配独立 API Key
- 使用统计和成本分析
- 支持 Claude Code / Codex / Gemini CLI / Droid CLI

**安装**:
```bash
curl -fsSL https://pincc.ai/manage.sh -o manage.sh && chmod +x manage.sh && ./manage.sh install
```

**特点**: 需要自己搭建服务器，适合多人共享分摊成本。

**适合**: 想和朋友拼车分摊订阅的用户。

---

## 功能对比

| 功能 | CC-Switch | Antigravity | zcf | CCR | CRS |
|------|-----------|-------------|-----|-----|-----|
| Provider 管理 | ✅ | ✅ | ✅ | ✅ | ✅ |
| MCP 管理 | ✅ | ❓ | ❓ | ❌ | ❌ |
| Skills 管理 | ✅ | ❓ | ✅ | ❌ | ❌ |
| Prompts 管理 | ✅ | ❓ | ✅ | ❌ | ❌ |
| 协议代理 | ❌ | ✅ | ✅ | ✅ | ✅ |
| 零配置安装 | ❌ | ❌ | ✅ | ❌ | ❌ |
| CLI 版本 | ✅ | ❓ | ✅ | ✅ | ❌ |
| 图形界面 | ✅ | ✅ | ❌ | ❌ | ✅ |
| 多人共享 | ❌ | ❌ | ❌ | ❌ | ✅ |
| 自建服务 | ❌ | ❌ | ❌ | ❌ | ✅ |

---

## 选择建议

- **最全面的配置管理** → CC-Switch
- **协议代理 + 多账号** → Antigravity-Manager
- **新手快速上手** → zcf
- **简单路由** → CCR
- **多人拼车共享** → CRS

---

## 链接

- CC-Switch: https://github.com/farion1231/cc-switch
- Antigravity-Manager: https://github.com/lbjlaq/Antigravity-Manager
- zcf: https://github.com/UfoMiao/zcf
- CCR: https://github.com/musistudio/claude-code-router
- CRS: https://github.com/Wei-Shaw/claude-relay-service