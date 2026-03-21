---
title: "系统拆解 CDP 和 Playwright：浏览器自动化到底该站在哪一层"
date: 2026-03-21
categories: [AI]
tags: [cdp, playwright, chromium, browser-automation, testing, agent]
---

> 很多人第一次接触浏览器自动化时，会把 CDP 和 Playwright 混成一件事。  
> 但这两者其实根本不在同一层。

如果只用一句话概括：

- **CDP** 是浏览器暴露出来的**底层调试/控制协议**
- **Playwright** 是构建在浏览器能力之上的**高层自动化框架**

这篇文章想系统讲清 5 个问题：

1. CDP 到底是什么  
2. Playwright 到底解决了什么问题  
3. 两者是什么关系  
4. 什么时候该直接用 CDP，什么时候该上 Playwright  
5. 在 AI Agent 语境下，为什么大多数人最后会更偏向 Playwright，而不是直接裸连 CDP

## 为什么这个区分重要

“我想控制浏览器”这句话，实际上可能对应三类完全不同的问题：

- 我要拿到底层网络、性能、Target、Browser domain 这类原始能力
- 我要稳定地点页面、填表单、等页面 ready、跑测试
- 我要让一个 AI Agent 去理解并操作网页

这三类问题的最优抽象层并不一样。

很多人一开始看到 Chrome DevTools Protocol，觉得这就是浏览器自动化的“真身”；也有人反过来以为 Playwright 只是一个更方便的 CDP 封装。两种理解都不准确。

更准确的看法是：

**CDP 解决的是“浏览器愿意把哪些能力开放出来”；Playwright 解决的是“人和程序怎样才能稳定地用这些能力完成任务”。**

## 第一层：CDP 是什么

CDP，全称 **Chrome DevTools Protocol**。官方定义非常直接：它允许外部工具对 Chromium/Chrome/Blink 系浏览器进行 instrument、inspect、debug 和 profile。协议本身由一组 domain 组成，比如 `DOM`、`Network`、`Debugger`、`Page`、`Target`、`Browser` 等。[官方协议文档](https://chromedevtools.github.io/devtools-protocol/)

也就是说，CDP 的本质不是“自动化测试框架”，而是：

**浏览器把自己的内部控制面暴露成了一套 JSON-RPC 风格的远程协议。**

### CDP 的核心结构

CDP 最核心的几个概念是：

- **Domain**：能力分区，比如 `Network`、`Page`、`Runtime`
- **Command**：你发给浏览器的操作
- **Event**：浏览器异步推给你的通知
- **Target**：被调试对象，比如 page、iframe、worker、browser 本身
- **Session**：你附着到某个 target 后形成的会话

官方的 `Target` domain 文档写得很清楚：它负责 target 发现、附着、创建页面、创建 browser context 等。[Target domain](https://chromedevtools.github.io/devtools-protocol/tot/Target/)

从连接方式看，CDP 常见入口是：

- 启动 Chrome 时打开 `--remote-debugging-port`
- 再访问：
  - `/json/version`
  - `/json/protocol`
  - `ws://.../devtools/browser/...`
  - `ws://.../devtools/page/...`

官方 FAQ 明确写到：

- `webSocketDebuggerUrl` 会出现在 `/json/version`
- `/json/protocol` 可以拿到当前浏览器支持的完整协议定义  
  来源：[CDP FAQ](https://chromedevtools.github.io/devtools-protocol/)

### CDP 适合做什么

如果你直接用 CDP，最自然的应用场景通常是：

- DevTools 类工具
- 浏览器性能分析
- 网络抓包和调试
- JS runtime / debugger
- 更底层的浏览器控制
- 自定义 browser instrumentation

比如 `Network` domain 就支持：

- 监听请求/响应
- 读响应体
- 设置 cookie
- 改 header
- 控制 cache / service worker / user agent  
  来源：[Network domain](https://chromedevtools.github.io/devtools-protocol/tot/Network/)

所以从“能力面”来说，CDP 很强。

但它的问题也很明显：

### CDP 的问题：能力强，不等于工作流好

CDP 是协议，不是任务模型。

协议层通常不会替你处理这些高频麻烦：

- 元素是不是已经稳定可点击
- 动画是不是还没结束
- 重绘后之前的节点句柄是不是失效了
- 多 page / popup / iframe / worker 的状态如何协调
- 同一个操作失败了该不该重试
- 最终该怎样形成稳定的测试和调试工作流

换句话说，CDP 给你的是**零件**，不是**整机**。

如果直接用 CDP 写浏览器自动化，通常很快会遇到三个问题：

1. **语义过底**
   你拿到的是协议命令，而不是“一个人类真实要做的动作”

2. **Chromium 绑定**
   CDP 本质上是 Chrome/Chromium 世界的协议，不是跨浏览器标准

3. **稳定性工作全要自己做**
   等待、重试、定位、可操作性检查、调试追踪，基本都要自己补

## 第二层：Playwright 是什么

Playwright 官方定位很清楚：它是一个面向 **Chromium、WebKit 和 Firefox** 的自动化框架，而不是 Chromium-only 工具。[Browsers](https://playwright.dev/docs/browsers)

这点非常关键。

如果你把 Playwright 理解成“CDP 客户端”，第一步就错了。因为：

- CDP 是 Chromium 系协议
- Playwright 是跨 Chromium / Firefox / WebKit 的统一自动化抽象

这已经说明它的职责不是“原样转发 CDP”，而是提供一层**更高阶、更稳定、更跨浏览器的自动化语义**。

### Playwright 到底多做了什么

Playwright 真正解决的，不是“能不能控制浏览器”，而是：

**怎样把浏览器操作变成一个足够稳定的工程接口。**

它最核心的能力主要有几类。

### 1. Browser / Context / Page 的高层模型

Playwright 把浏览器组织成一套很清晰的对象层次：

- `Browser`
- `BrowserContext`
- `Page`

官方文档明确说，一个 `BrowserContext` 可以有多个 `Page`，而 context 级别还能共享 locale、viewport、network routes 等。[Pages](https://playwright.dev/docs/pages)

这比你自己在 CDP 里手动管理 target/session 更像真正的编程接口。

### 2. Locator 模型

Playwright 最重要的设计之一就是 locator。

官方文档写得很直接：

> Locators are the central piece of Playwright's auto-waiting and retry-ability.

[Locators](https://playwright.dev/docs/locators)

这句话非常重要。它意味着 Playwright 并不是“帮你把 selector 包一下”，而是在说：

**元素定位本身就是自动等待、重试和稳定性交互的中心抽象。**

比如官方推荐优先使用：

- `getByRole`
- `getByLabel`
- `getByText`
- `getByPlaceholder`

而不是长 CSS / XPath 链。

### 3. Auto-waiting / Actionability checks

这基本是 Playwright 和裸 CDP 体验差异最大的地方。

官方文档写得很清楚，`locator.click()` 前会自动检查：

- locator 是否唯一
- 元素是否可见
- 是否稳定
- 是否能接收事件
- 是否 enabled  
  来源：[Auto-waiting](https://playwright.dev/docs/actionability)

这就不是底层协议层会替你做的事情了。

这也是为什么很多“自己写浏览器驱动”的系统一开始看起来能跑，但很快会陷入 flaky 地狱。  
不是因为它不会点按钮，而是因为它不会在正确的时机、以稳定的方式点按钮。

### 4. Trace、Report、Inspector

Playwright 不是只有执行层，还有调试层。

官方文档里 trace viewer 的描述很准确：你可以按 action 回放，查看页面前后状态、log、source、network、error、console，而且 trace viewer 还会创建 DOM snapshot。[Trace Viewer](https://playwright.dev/docs/trace-viewer-intro)

这意味着 Playwright 除了“能跑”，还很强调：

- 出错后怎么定位
- 测试失败后怎么回放
- 自动化行为怎么审计

### 5. 跨浏览器

这是 Playwright 和 CDP 最本质的结构性区别。

官方文档写得很明确：Playwright 支持 Chromium、Firefox、WebKit，也支持 branded browsers 如 Chrome 和 Edge。[Browsers](https://playwright.dev/docs/browsers)

这意味着它的核心价值不是接入某一个浏览器的某一套调试协议，而是：

**把“自动化浏览器”这件事抽象成一个跨内核工作流。**

## 第三层：CDP 和 Playwright 到底是什么关系

可以把它们理解成这样：

```text
应用/测试/Agent
        │
        ▼
   Playwright
        │
        ├─ Chromium 路径：必要时可接入 CDP
        ├─ Firefox 路径：Playwright 自己的适配层
        └─ WebKit 路径：Playwright 自己的适配层
```

所以不是：

```text
Playwright = CDP 的别名
```

而更像：

```text
CDP 是 Playwright 在 Chromium 世界里可以利用的一部分能力接口
```

### Playwright 可以直接暴露 CDP

这点官方也写得很明确。

Playwright 提供了 `CDPSession`：

- `session.send(method, params)`
- `session.on(event, ...)`

官方定义就是：

> The CDPSession instances are used to talk raw Chrome Devtools Protocol.

[CDPSession](https://playwright.dev/docs/api/class-cdpsession)

也就是说，如果你平时用 Playwright，但某个场景必须下探到底层 CDP domain，你可以直接开 `newCDPSession(page)`。

这是一个非常实用的混合模式：

- 日常交互用 Playwright 高层 API
- 特殊能力用原始 CDP

### Playwright 也支持直接 attach 到现有 CDP 端点

官方 `browserType.connectOverCDP()` 文档说明：

- 只支持 Chromium-based browsers
- 它是通过 CDP 把 Playwright attach 到现有浏览器实例
- 但官方明确提醒：

> This connection is significantly lower fidelity than the Playwright protocol connection via browserType.connect().

[BrowserType.connectOverCDP](https://playwright.dev/docs/api/class-browsertype)

这句话很关键。

它说明了两件事：

1. Playwright 确实可以“借 CDP 接进来”
2. 但这不是它最完整、最强的工作模式

换句话说，**CDP attach 是兼容入口，不是 Playwright 的最佳语义入口。**

### Playwright 有时也会借用 CDP 作为连接通道

官方 Selenium Grid 文档里甚至直接写了：

> Internally, Playwright connects to the browser using Chrome DevTools Protocol websocket.

这个描述出现在 Selenium Grid 场景下。[Selenium Grid](https://playwright.dev/docs/selenium-grid)

但这里不要误解成“Playwright 本质上就是 CDP 客户端”。更准确的说法是：

**在某些 Chromium 集成场景里，Playwright 会借 CDP 作为 transport 或兼容入口；但 Playwright 的产品价值并不等于这个 transport。**

## 第四层：什么时候该直接用 CDP

直接用 CDP，适合这几类问题：

### 1. 你要做的是浏览器基础设施或底层工具

比如：

- 自定义调试器
- 网络/性能分析器
- 浏览器观测平台
- DevTools 类功能
- 安全研究 / 内核观测 / instrumentation

这时候你需要的是“原始能力”，不是“更顺手的点击 API”。

### 2. 你明确只服务 Chromium

如果目标环境就是 Chrome/Chromium/Edge/WebView2，而不是跨浏览器测试，直接使用 CDP 的限制会小很多。

### 3. 你需要 Playwright 没有直接暴露的 domain

比如某些浏览器内部能力、实验性 domain、或者你要直接订阅原始协议事件流。

这时最常见的做法是：

- 要么直接写 CDP 客户端
- 要么 Playwright + `CDPSession`

## 第五层：什么时候该优先用 Playwright

如果你的目标是下面这些，默认应该优先考虑 Playwright：

### 1. E2E 测试

这是它最典型的战场。  
locator、auto-wait、trace、report、auth state、browser context 隔离，这些都是测试框架级能力。

### 2. 稳定的浏览器自动化

例如：

- 登录
- 表单填写
- 多页面流程
- 文件上传
- API mock
- WebSocket mock/intercept

这些任务真正的痛点不是“怎么发命令”，而是“怎么别 flaky”。

### 3. 跨浏览器验证

只要你需要同时覆盖 Chromium / Firefox / WebKit，CDP 直接出局。

### 4. AI Agent 操作网页

这一点在 2025-2026 年变得越来越重要。

AI Agent 并不擅长管理一堆底层协议细节。模型更适合的接口通常是：

- 打开页面
- 找到元素
- 等待可交互
- 点击
- 填写
- 断言
- 读取网络和 trace

也就是说，**Agent 更适合消费“动作语义”，而不是“协议语义”。**

这也是为什么你会看到越来越多 agent/browser 工具最终都往 Playwright 这类高层接口靠，而不是让模型直接裸写 CDP 命令。

## 第六层：最实用的选择，其实是混合模式

现实里最常见、也最合理的模式不是二选一，而是：

### 默认用 Playwright

处理：

- 页面导航
- 元素定位
- 表单交互
- 等待
- 断言
- 调试和回放

### 遇到边角需求时，下探 CDP

处理：

- 特定 Chromium-only 能力
- 实验性 domain
- 更底层的 browser/page instrumentation

示意代码大概像这样：

```ts
import { chromium } from 'playwright';

const browser = await chromium.launch();
const page = await browser.newPage();
await page.goto('https://example.com');

const cdp = await page.context().newCDPSession(page);
await cdp.send('Network.enable');
await cdp.send('Network.setCacheDisabled', { cacheDisabled: true });
```

这个组合的价值在于：

- 绝大多数复杂度由 Playwright 吃掉
- 只有真的需要时才碰 CDP

## 第七层：几个常见误区

### 误区 1：CDP 更底层，所以一定更强

不一定。

CDP 的**能力面**更原始，但在“完成一个真实自动化任务”这件事上，Playwright 往往更强，因为它把等待、稳定性、定位和调试工作流都做掉了。

### 误区 2：Playwright 只是 CDP 的语法糖

不对。

如果它只是 CDP 语法糖，就不可能天然支持 Firefox 和 WebKit，也不可能形成 locator / actionability / trace 这一整套独立工作流。

### 误区 3：只要能 `connectOverCDP`，就等于得到了完整 Playwright

官方文档已经明确说了，`connectOverCDP()` 是 **significantly lower fidelity** 的连接方式。  
适合兼容接入，不适合把它当最佳模式。

### 误区 4：remote debugging port 永远都能随便开

这在新版本 Chrome 上已经不是理所当然了。

Chrome 官方在 2025 年的安全更新里明确说：

- 从 Chrome 136 开始
- `--remote-debugging-port` 和 `--remote-debugging-pipe`
- 对默认 Chrome 数据目录不再照旧生效
- 需要配合非默认 `--user-data-dir`

而且官方建议浏览器自动化场景优先用 **Chrome for Testing**。  
来源：[Chrome remote debugging security change](https://developer.chrome.com/blog/remote-debugging-port)

这对所有“直接依赖 CDP attach 真实用户 Chrome”的系统都很重要。

## 总结

如果要把这篇文章压成一句话，我会这么说：

**CDP 是浏览器的底层控制协议，Playwright 是把浏览器自动化做成稳定工程接口的高层框架。**

所以真正的选型问题不是“哪个更高级”，而是：

- 你是在做**浏览器能力开发**，还是做**浏览器任务执行**
- 你是在要**原始协议控制面**，还是要**可维护的自动化工作流**

我自己的经验判断是：

- 做 DevTools、性能工具、浏览器基础设施：优先看 CDP
- 做测试、爬交互页面、agent 浏览器操作：优先看 Playwright
- 做 Chromium-only 的高级自动化：Playwright 为主，必要时补 `CDPSession`

这也是最不容易把系统做歪的路径。

## 参考资料

- [Chrome DevTools Protocol 官方文档](https://chromedevtools.github.io/devtools-protocol/)
- [CDP Target Domain](https://chromedevtools.github.io/devtools-protocol/tot/Target/)
- [CDP Network Domain](https://chromedevtools.github.io/devtools-protocol/tot/Network/)
- [Playwright Browsers 文档](https://playwright.dev/docs/browsers)
- [Playwright BrowserType.connectOverCDP](https://playwright.dev/docs/api/class-browsertype)
- [Playwright CDPSession](https://playwright.dev/docs/api/class-cdpsession)
- [Playwright Auto-waiting](https://playwright.dev/docs/actionability)
- [Playwright Locators](https://playwright.dev/docs/locators)
- [Playwright Trace Viewer](https://playwright.dev/docs/trace-viewer-intro)
- [Playwright Selenium Grid](https://playwright.dev/docs/selenium-grid)
- [Chrome 136 remote debugging 安全变更](https://developer.chrome.com/blog/remote-debugging-port)
