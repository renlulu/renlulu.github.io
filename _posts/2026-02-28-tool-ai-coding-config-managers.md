---
title: "【工具】AI Coding 配置管理工具对比：CC-Switch vs Antigravity vs zcf vs CCR"
date: 2026-02-28 21:00:00 +0800
categories: [Tools]
tags: [claude-code, codex, gemini, tools, comparison]
---

> Claude Code、Codex、Gemini CLI 火了，配套的配置管理工具也涌现出来。CC-Switch、Antigravity-Manager、zcf、CCR 都解决类似的问题，但定位不同。本文对比这四款工具。

## 为什么需要这些工具？

如果你使用 Claude Code、Codex、Gemini CLI，会遇到这些问题：

1. **API 配置切换麻烦** — 官方 API、中转服务、自定义 endpoint
2. **多账号管理** — 不同账号、不同额度
3. **成本控制** — 官方贵，中转便宜
4. **配置分散** — 每个工具都有自己的配置文件

这些工具就是为了解决这些问题。

---

## 四款工具概览

| 工具 | Stars | 定位 | 技术栈 |
|------|-------|------|--------|
| **CC-Switch** | 21.9k | 一站式配置管理 | Tauri (Rust + React) |
| **Antigravity-Manager** | 24.8k | 协议代理 + 账号管理 | Tauri (Rust + React) |
| **zcf** | 5.5k | 零配置快速启动 | Node.js |
| **CCR** | ~257 | Claude Code Router | TypeScript |

---

## CC-Switch

**GitHub**: https://github.com/farion1231/cc-switch

**定位**: 一站式配置管理

**核心功能**:
- Provider 管理（一键切换 API 配置）
- MCP Server 统一管理
- Skills 管理（从 GitHub 自动扫描安装）
- Prompts 管理（系统提示词预设）
- 配置导入/导出/备份

**特点**: 功能最全面，覆盖 Provider/MCP/Skills/Prompts

**适合**: 想要全面管理配置的用户

---

## Antigravity-Manager

**GitHub**: https://github.com/lbjlaq/Antigravity-Manager

**定位**: 专业级 AI 账号管理与协议代理系统

**核心功能**:
- 多账号管理
- 协议转换（Web Session → API）
- 智能请求调度
- 负载均衡

**特点**: 偏向"协议代理"，将 Web Session 转为标准 API

**适合**: 需要高性能 API 中转的用户

**备注**: 看起来是 "Antigravity Tools" 这个产品的配套工具

---

## zcf

**GitHub**: https://github.com/UfoMiao/zcf

**定位**: Zero-Config Code Flow，零配置快速启动

**核心功能**:
- 一键安装配置
- workflows 自动生成
- API/CCR 快速接入
- MCP 自动配置

**特点**: 强调"零配置"，新手友好

**安装**:
```bash
npx zcf i  # 全自动安装
```

**适合**: 想要快速上手的用户

---

## CCR

**GitHub**: https://github.com/musistudio/claude-code-router

**定位**: Claude Code Router，代理/路由

**核心功能**:
- 多 provider 路由
- API 代理
- 请求转发

**特点**: 专注于"路由"功能，轻量级

**适合**: 需要简单代理的开发者

---

## 功能对比

| 功能 | CC-Switch | Antigravity | zcf | CCR |
|------|-----------|-------------|-----|-----|
| Provider 管理 | ✅ | ✅ | ✅ | ✅ |
| MCP 管理 | ✅ | ❓ | ❓ | ❌ |
| Skills 管理 | ✅ | ❓ | ✅ | ❌ |
| Prompts 管理 | ✅ | ❓ | ✅ | ❌ |
| 协议代理 | ❌ | ✅ | ✅ | ✅ |
| 零配置安装 | ❌ | ❌ | ✅ | ❌ |
| CLI 版本 | ✅ | ❓ | ✅ | ✅ |
| 图形界面 | ✅ | ✅ | ❌ | ❌ |

---

## 选择建议

**如果你想要最全面的配置管理** → CC-Switch

**如果你需要协议代理 + 多账号** → Antigravity-Manager

**如果你是新手，想快速上手** → zcf

**如果你只需要简单路由** → CCR

---

## 链接

- CC-Switch: https://github.com/farion1231/cc-switch
- Antigravity-Manager: https://github.com/lbjlaq/Antigravity-Manager
- zcf: https://github.com/UfoMiao/zcf
- CCR: https://github.com/musistudio/claude-code-router