---
title: "Vitalik 推翻了自己：以太坊 Rollup-Centric 路线图宣告终结"
date: 2026-02-28 20:00:00 +0800
categories: [区块链]
tags: [Ethereum, Vitalik, L1, L2, Scaling, ZK-EVM]
---

2026 年 2 月，Vitalik Buterin 在 X 上连发多条长线程，核心信息只有一个：

> **"The original vision of L2s and their role in Ethereum no longer makes sense."**

以太坊坚持了好几年的 rollup-centric roadmap，被创始人自己亲手翻篇了。

## 背景：rollup-centric 路线图是什么

2020 年 10 月，Vitalik 发表了 [A rollup-centric ethereum roadmap](https://ethereum-magicians.org/t/a-rollup-centric-ethereum-roadmap/4698)，定下了以太坊的扩展基调：**L1 做共识和数据可用性，扩展交给 L2 rollup**。

这个方向影响了整个生态：Optimism、Arbitrum、zkSync、StarkNet、Base…… 无数 L2 项目拔地而起，OP Stack 和 ZK Stack 成了基础设施标配。"L2 是以太坊的品牌分片（branded shard）"，几乎是共识。

然后现实打了脸。

## 两个残酷的事实

Vitalik 在[第一条线程](https://x.com/VitalikButerin/status/2018711006394843585)中直接摊牌，指出两个事实：

### 1. L2 去中心化进展远慢于预期

大部分 L2 至今仍停留在 Stage 0 或 Stage 1。所谓 Stage 2（完全去中心化，证明系统可信，升级需延时）几乎没有项目达到。Vitalik 甚至点出一个尖锐的原因：

> "Their customers' regulatory needs require them to have ultimate control."

翻译：有些 L2 不是不能去中心化，是**不想**。客户需要它们保持控制权来满足合规要求。这从根本上违背了"品牌分片"的定位——分片是没有管理员的。

互操作性（interop）也一样，各 L2 之间仍然是孤岛，跨 L2 体验远不如在同一条链上操作。

### 2. L1 本身在快速扩展

2025 年以太坊 gas limit 从 30M 翻倍到 60M，PeerDAS 上线大幅提升了 blob 容量。L1 的 gas 费已经很低了，而且 2026 年还会继续大幅提升 gas limit。

当 L1 本身已经够便宜、够快的时候，为什么还要把用户推到一个去中心化打折扣、互操作性拉胯的 L2 上？

## 新方向：L2 不再是扩展方案

Vitalik 给出了新定位：

> L2 应该 **"identify a value-add other than scaling"**。

不要再卖"我是以太坊的扩展层"这个故事了。L2 应该做的是**差异化**：

| 方向 | 说明 |
|------|------|
| 隐私 | L1 做不到的链上隐私交易 |
| 专用 VM | 非 EVM 的执行环境（Move、Cairo 等） |
| 极低延迟 | 游戏、高频交易需要的亚秒级确认 |
| 应用特定链 | 社交、身份、AI 等垂直场景 |
| 不同信任模型 | 让用户自己选择信任级别 |

L2 不再是"以太坊官方扩展层"，而是一个**完整的光谱**——从高度依赖以太坊安全性到相对独立，各取所需。

技术上，Vitalik 提出了 **Native Rollup Precompile**：把 ZK-EVM 证明验证直接做进以太坊协议，这样 rollup 可以自动跟随协议升级，实现无信任互操作和同步可组合性。这算是给 L2 留了一条"回归正统"的路。

## 短期扩展：Glamsterdam 升级

在[第二条线程](https://x.com/VitalikButerin/status/2027403360484430122)中，Vitalik 详细拆解了 L1 扩展计划，分短期和长期两个桶。

短期的核心都在 **Glamsterdam 升级**中：

### Block-Level Access Lists（区块级访问列表）

当前以太坊验证区块是**串行**的——一笔交易接一笔交易执行。Block-level access lists 让节点预先知道每笔交易会访问哪些状态，从而可以**并行验证**区块的不同部分。

### ePBS（Enshrined Proposer-Builder Separation）

目前的 PBS 是通过 MEV-Boost 等外部中继实现的。ePBS 把它写入协议本身，带来的好处之一是：可以安全地使用 12 秒 slot 中的**大部分时间**来验证区块。

当前为了安全，验证只用了几百毫秒就结束了，剩下的时间浪费了。ePBS 让这部分时间可以被利用起来，等于同样的 slot 时间能处理更多交易。

### Gas 重定价 + 多维度 Gas

Gas 成本应该反映**实际执行成本**。当前有些操作的 gas 定价与真实计算量不匹配。重定价修正这些偏差。

更重要的是**多维度 Gas**：在 Glamsterdam 中，"状态创建"和"执行 + calldata"的成本被**分离**。

为什么这很重要？因为：
- **执行**是一次性的，验证完就结束了
- **状态创建**是永久性的，每个节点都要永远存储

把它们混在一起定价，意味着要么执行太贵（限制了吞吐量），要么状态创建太便宜（链膨胀）。分开定价后，执行容量可以更激进地增长，同时对状态膨胀保持严格限制。

## 长期扩展：ZK-EVM + Blob

### ZK-EVM

这是最根本的变革。当前以太坊的验证模型是：**每个节点重新执行每笔交易**。这是扩展的瓶颈——区块越大，节点负担越重。

ZK-EVM 的愿景：出块者执行交易并生成零知识证明，验证者只需**验证证明**而不用重新执行。验证证明的成本远低于执行本身。

时间线：
- **2026 年**：有限可用，部分客户端可以用 ZK-EVM 参与验证
- **2027 年**：更广泛实现，solo staker 可以通过 ZK 证明廉价验证

这意味着未来即使区块容量增加 10-100 倍，普通硬件的节点仍然可以参与验证。去中心化不用为扩展让步。

### Blob 扩容

继续迭代 PeerDAS，目标是支持 **~8 MB/s** 的数据吞吐。Blob 原本是给 L2 提交数据用的，但未来以太坊 L1 自己的交易数据也可以放进 blob——进一步降低 L1 的负担。

## Hyper-scaling 状态：最难的部分

在[第三条线程](https://x.com/VitalikButerin/status/2019437232315056267)中，Vitalik 坦承了一个难题：

> "We want 1000x scale on Ethereum L1. We roughly know how to do this for execution and data. But **scaling state is fundamentally harder**."

执行可以用 ZK 证明搞定，数据可以用 blob 搞定，但**状态**——就是链上所有账户余额、合约存储、代码——没法这么简单压缩。每个全节点都需要维护完整状态，这是个持续增长的负担。

Vitalik 的思路是**创建新的状态形式**来解决这个问题。具体方案还在研究中，但方向是明确的：通过重新设计状态存储的方式，而不是简单粗暴地增加硬件要求。

## Strawmap：七次硬分叉到 2029

这一切的宏观框架是 Ethereum Foundation 研究员 Justin Drake 提出的 [Strawmap](https://www.coindesk.com/tech/2026/02/26/ethereum-foundation-drops-most-ambitious-roadmap-in-years-targets-finality-in-seconds-by-2029)（strawman + roadmap），规划了从现在到 2029 年的**七次硬分叉**，五个"北极星"目标：

| 目标 | 说明 |
|------|------|
| **Fast L1** | Slot 时间从 12 秒逐步降到 2 秒，finality 从 ~16 分钟降到 6-16 秒 |
| **Gigagas L1** | L1 吞吐量达到 gigagas 级别 |
| **Teragas L2** | L2 的总吞吐量达到 teragas 级别 |
| **Post-Quantum** | 抗量子计算攻击 |
| **Native Privacy** | 协议层原生隐私 |

其中 finality 的改进依赖一个叫 **Minimmit** 的单轮 BFT 算法，可以在 6-16 秒内完成最终确认。Slot 时间的缩短则会渐进式进行，Vitalik 建议按 "每次缩短 √2 倍" 的节奏推进。

## 总结

这不是一次小修补，而是以太坊扩展哲学的根本转向：

1. **L2 从"必需品"变成"可选品"**——L1 自己扩展，L2 去做差异化
2. **执行模型从"全部重新执行"变成"验证证明"**——ZK-EVM 是核心
3. **Gas 从一维变成多维**——执行和状态分开定价，各自独立扩展
4. **共识从慢到快**——12 秒 slot → 2 秒 slot，16 分钟 finality → 秒级

对于 L2 项目方来说，这是一个明确的信号：不要再躺在"以太坊扩展层"的叙事上了。找到自己不可替代的价值，否则 L1 扩展起来以后，纯粹的"便宜版以太坊"没有存在的意义。

对于整个生态来说，Vitalik 亲手推翻自己六年前定下的路线图，既需要勇气，也说明以太坊的方向还没有固化。无论你看好还是看空 ETH，这个生态至少还在认真思考和迭代。

---

*本文基于 Vitalik Buterin 2026 年 2 月在 X 上发布的多条推文线程整理。原始来源：[线程1](https://x.com/VitalikButerin/status/2018711006394843585)、[线程2](https://x.com/VitalikButerin/status/2027403360484430122)、[线程3](https://x.com/VitalikButerin/status/2019437232315056267)*
