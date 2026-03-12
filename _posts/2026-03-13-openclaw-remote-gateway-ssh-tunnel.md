---
title: 给远程 OpenClaw Gateway 打洞：SSH 隧道 + Chrome Extension 实战
date: 2026-03-13
categories: [DevOps]
tags: [openclaw, ssh-tunnel, gcp, chrome-extension]
---

## 背景

我在 GCP Compute Engine 上跑了一个 OpenClaw agent，想在本地通过 Chrome Extension 控制它的浏览器工具。问题是：

- VM **没有外部 IP**（只有内网 IP）
- Gateway 监听在 `0.0.0.0:8080`（`--bind lan`）
- `openclaw node run` 连外部 IP 时要求 `wss://`（TLS），直接连会报安全错误
- VM 使用 GCP OS Login，不能用普通 SSH 密钥直连

目标：让本地的 `openclaw node run` 能连上远程 Gateway，从而让 Chrome Extension 工作。

## 方案概览

```
本地 Chrome Extension
        ↕
本地 openclaw node host (127.0.0.1:18789)
        ↕  ← 这一段需要打洞
远程 VM Gateway (内网:8080)
```

最终方案：**给 VM 加外部 IP + 环境变量绕过 TLS 检查**。中间踩了不少坑，记录一下。

## 第一次尝试：SSH 隧道

### 思路

通过 `gcloud compute ssh` 的端口转发，把本地 18789 映射到 VM 的 8080：

```bash
gcloud compute ssh <vm-name> --zone=us-west1-a -- -NL 18789:localhost:8080
```

然后本地连 loopback：

```bash
openclaw node run --host 127.0.0.1 --port 18789
```

### 踩坑

1. **`-L 8080:127.0.0.1:8080` 连不上**：Gateway 用 `--bind lan` 启动，虽然实际绑在 `0.0.0.0`，但第一次隧道目标写了 `127.0.0.1` 没通。改成 `-L 18789:localhost:8080` 后解决。

2. **本地端口被占**：之前挂起的 SSH session（Ctrl+Z）占了端口，新隧道绑不上。需要先 kill 掉旧进程。

3. **device signature invalid**：隧道通了，但 `openclaw node run` 报 device signature 校验失败。清理了两端的 `~/.openclaw/devices/` 和 `~/.openclaw/identity/` 目录都没用。怀疑是 SSH 隧道对 WebSocket 握手有影响。

## 最终方案：外部 IP + Insecure 模式

### 1. 给 VM 加外部 IP

```bash
gcloud compute instances add-access-config <vm-name> \
  --zone=us-west1-a \
  --access-config-name="external-nat"
```

确认防火墙规则已经开放了 `tcp:8080`（target tag 匹配 VM 的网络标签）。

### 2. 配置 Gateway Auth Token

在 OpenClaw 的配置文件中加入 token 认证：

```json
{
  "gateway": {
    "auth": {
      "mode": "token",
      "token": "<your-gateway-token>"
    },
    "controlUi": {
      "dangerouslyDisableDeviceAuth": true
    }
  }
}
```

如果用 entrypoint.sh 自动生成配置，确保 `OPENCLAW_GATEWAY_TOKEN` 环境变量被写入 auth 配置。

### 3. 启动 Node Host

直连外部 IP，用环境变量绕过 TLS 和 token 认证：

```bash
OPENCLAW_ALLOW_INSECURE_PRIVATE_WS=1 \
OPENCLAW_GATEWAY_TOKEN=<your-gateway-token> \
openclaw node run --host <vm-external-ip> --port 8080
```

关键环境变量：
- `OPENCLAW_ALLOW_INSECURE_PRIVATE_WS=1`：允许 `ws://` 明文连接（默认只允许 `wss://`）
- `OPENCLAW_GATEWAY_TOKEN`：Gateway 的认证 token

### 4. 安装 Chrome Extension

```bash
openclaw browser extension install
```

然后在 `chrome://extensions` 开启开发者模式，点 **Load unpacked** 加载输出的目录。

## 踩坑总结

| 问题 | 原因 | 解决 |
|------|------|------|
| `ECONNREFUSED 127.0.0.1:8080` | SSH 隧道没绑上本地端口（旧 session 占用） | kill 旧进程，换端口 |
| `SECURITY ERROR: Cannot connect over plaintext ws://` | openclaw 默认要求 wss:// | `OPENCLAW_ALLOW_INSECURE_PRIVATE_WS=1` |
| `gateway token missing` | Gateway 配了 token auth 但客户端没带 | `OPENCLAW_GATEWAY_TOKEN=<token>` |
| `device signature invalid` | SSH 隧道下 WebSocket 握手异常 | 改用外部 IP 直连 |
| `--token` 参数不存在 | `openclaw node run` 不支持 `--token` flag | 用环境变量 `OPENCLAW_GATEWAY_TOKEN` |
| `Permission denied (publickey)` | VM 用 OS Login，不能普通 ssh 直连 | 必须用 `gcloud compute ssh` |

## 安全提示

- `OPENCLAW_ALLOW_INSECURE_PRIVATE_WS=1` 会让连接走明文，只在可信网络使用
- 生产环境应该用 TLS（Caddy 反向代理 / 子域名 + 通配符证书）
- Gateway token 不要硬编码在脚本里，建议用环境变量或 dotenv 管理
