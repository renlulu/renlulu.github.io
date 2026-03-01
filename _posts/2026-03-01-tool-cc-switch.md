---
title: "【工具】CC-Switch：Claude Code / Codex / Gemini 一站式管理"
date: 2026-02-28 20:00:00 +0800
categories: [Tools]
tags: [claude-code, codex, gemini, mcp, tools]
---

> 一个跨平台桌面应用，统一管理 Claude Code、Codex、Gemini CLI 的 Provider、MCP、Skills 和 Prompts。21.9k stars，解决的是"配置管理"的真实痛点。

## 解决什么问题？

如果你同时使用 Claude Code、Codex、Gemini CLI，会遇到这些问题：

1. **Provider 切换麻烦** — 官方 API、中转服务、自定义 endpoint，要手动改配置文件
2. **MCP Server 分散** — 三个应用各自有配置格式，不统一
3. **Skills 发现困难** — 不知道有哪些可用，手动安装麻烦
4. **Prompts 管理** — 不同场景需要不同提示词，切换不方便

CC-Switch 把这些全部统一管理。

## 核心功能

| 功能 | 说明 |
|------|------|
| **Provider 管理** | 一键切换 API 配置，支持官方 + 多个中转服务 |
| **MCP Server 管理** | 统一管理三个应用的 MCP，支持 stdio/http/sse |
| **Skills 管理** | 从 GitHub 仓库自动扫描、一键安装/卸载 |
| **Prompts 管理** | 系统提示词预设，Markdown 编辑器 |
| **配置同步** | 导入/导出、自动备份、WebDAV 同步（CLI 版） |

## 安装

**macOS (Homebrew):**

```bash
brew tap farion1231/ccswitch
brew install --cask cc-switch
```

**Linux:**

```bash
# Debian/Ubuntu
sudo dpkg -i CC-Switch-v3.11.1-Linux.deb

# AppImage
chmod +x CC-Switch-v3.11.1-Linux.AppImage
./CC-Switch-v3.11.1-Linux.AppImage
```

**Windows:**

下载 MSI 安装包或 Portable 版本。

## 架构设计

```
CC-Switch (SSOT)
~/.cc-switch/cc-switch.db
        ↓ 写入
┌─────────────────────────────────┐
│ Claude: ~/.claude/settings.json │
│         ~/.claude.json (MCP)    │
│ Codex:  ~/.codex/auth.json      │
│         ~/.codex/config.toml    │
│ Gemini: ~/.gemini/.env          │
│         ~/.gemini/settings.json │
└─────────────────────────────────┘
```

所有配置存储在 SQLite 数据库（SSOT），切换时自动同步到各应用的配置文件。

## CLI 版本

如果你更喜欢终端：

```bash
# 快速安装
curl -fsSL https://github.com/SaladDay/cc-switch-cli/releases/latest/download/install.sh | bash

# 使用
cc-switch provider list
cc-switch provider switch <id>
cc-switch mcp sync
```

CLI 版本还支持 WebDAV 同步。

## 链接

- GitHub: https://github.com/farion1231/cc-switch
- CLI 版本: https://github.com/SaladDay/cc-switch-cli
- Releases: https://github.com/farion1231/cc-switch/releases